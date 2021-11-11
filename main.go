/*
Copyright 2015 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

The original file is located at [1].
[1]: https://github.com/kubernetes/kube-state-metrics/blob/e859b280fcc2/main.go

The original source was adjusted to:
- support a store.Builder which uses a controller-runtime client instead of client-go.
- remove sharding functionality to reduce complexity for the initial implementation.
- use a custom options package.
- rename the application.
*/

package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"
	"time"

	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
	"k8s.io/kube-state-metrics/v2/pkg/allowdenylist"
	"k8s.io/kube-state-metrics/v2/pkg/util/proc"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha4"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	//+kubebuilder:scaffold:imports
	"github.com/daimler/cluster-api-state-metrics/pkg/metricshandler"
	casmoptions "github.com/daimler/cluster-api-state-metrics/pkg/options"
	"github.com/daimler/cluster-api-state-metrics/pkg/store"
)

var (
	scheme = runtime.NewScheme()

	// variables set during build
	buildDate string
	gitBranch string
	gitCommit string
	user      string
)

const (
	metricsPath = "/metrics"
	healthzPath = "/healthz"
)

// promLogger implements promhttp.Logger
type promLogger struct{}

func (pl promLogger) Println(v ...interface{}) {
	klog.Error(v...)
}

// promLogger implements the Logger interface
func (pl promLogger) Log(v ...interface{}) error {
	klog.Info(v...)
	return nil
}

func init() {
	klog.InitFlags(nil)

	_ = clientgoscheme.AddToScheme(scheme)
	_ = clusterv1.AddToScheme(scheme)
	_ = controlplanev1.AddToScheme(scheme)
	_ = apiextensionsv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	opts := casmoptions.NewOptions()
	opts.AddFlags()

	promLogger := promLogger{}

	ctx := context.Background()

	err := opts.Parse()
	if err != nil {
		klog.Fatalf("Error: %s", err)
	}

	if opts.Version {
		version.BuildDate = buildDate
		version.BuildUser = user
		version.Branch = gitBranch
		version.Revision = gitCommit
		fmt.Printf("%s\n", version.Print("cluster-api-state-metrics"))
		os.Exit(0)
	}

	if opts.Help {
		opts.Usage()
		os.Exit(0)
	}
	storeBuilder := store.NewBuilder()

	casmMetricsRegistry := prometheus.NewRegistry()
	casmMetricsRegistry.MustRegister(version.NewCollector("cluster_api_state_metrics"))
	durationVec := promauto.With(casmMetricsRegistry).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "http_request_duration_seconds",
			Help:        "A histogram of requests for cluster-api-state-metrics metrics handler.",
			Buckets:     prometheus.DefBuckets,
			ConstLabels: prometheus.Labels{"handler": "metrics"},
		}, []string{"method"},
	)
	storeBuilder.WithMetrics(casmMetricsRegistry)

	var resources []string
	if len(opts.Resources) == 0 {
		klog.Info("Using default resources")
		resources = casmoptions.DefaultResources.AsSlice()
	} else {
		klog.Infof("Using resources %s", opts.Resources.String())
		resources = opts.Resources.AsSlice()
	}

	if err := storeBuilder.WithEnabledResources(resources); err != nil {
		klog.Fatalf("Failed to set up resources: %v", err)
	}

	namespaces := opts.Namespaces.GetNamespaces()
	nsFieldSelector := namespaces.GetExcludeNSFieldSelector(opts.NamespacesDenylist)
	storeBuilder.WithNamespaces(namespaces, nsFieldSelector)

	allowDenyList, err := allowdenylist.New(opts.MetricAllowlist, opts.MetricDenylist)
	if err != nil {
		klog.Fatal(err)
	}

	err = allowDenyList.Parse()
	if err != nil {
		klog.Fatalf("error initializing the allowdeny list: %v", err)
	}

	klog.Infof("metric allow-denylisting: %v", allowDenyList.Status())

	storeBuilder.WithAllowDenyList(allowDenyList)

	storeBuilder.WithGenerateStoresFunc(storeBuilder.DefaultGenerateStoresFunc(), opts.UseAPIServerCache)

	proc.StartReaper()

	ctrlClient, err := getCtrlClient(opts.Kubeconfig, os.Getenv("KUBECONTEXT"))
	if err != nil {
		klog.Fatalf("Unable to get controller client: %v", err)
	}

	storeBuilder.WithCtrlClient(ctrlClient)
	storeBuilder.WithAllowAnnotations(opts.AnnotationsAllowList)
	storeBuilder.WithAllowLabels(opts.LabelsAllowList)

	casmMetricsRegistry.MustRegister(
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(),
	)

	var g run.Group

	m := metricshandler.New(
		ctx,
		opts,
		storeBuilder,
		opts.EnableGZIPEncoding,
	)

	tlsConfig := opts.TLSConfig

	telemetryMux := buildTelemetryServer(casmMetricsRegistry)
	telemetryListenAddress := net.JoinHostPort(opts.TelemetryHost, strconv.Itoa(opts.TelemetryPort))
	telemetryServer := http.Server{Handler: telemetryMux, Addr: telemetryListenAddress}

	metricsMux := buildMetricsServer(m, durationVec)
	metricsServerListenAddress := net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port))
	metricsServer := http.Server{Handler: metricsMux, Addr: metricsServerListenAddress}

	// Run Telemetry server
	{
		g.Add(func() error {
			klog.Infof("Starting cluster-api-state-metrics self metrics server: %s", telemetryListenAddress)
			return web.ListenAndServe(&telemetryServer, tlsConfig, promLogger)
		}, func(error) {
			ctxShutDown, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			if err := telemetryServer.Shutdown(ctxShutDown); err != nil {
				klog.ErrorS(err, "shutdown telemetry server")
			}
		})
	}
	// Run Metrics server
	{
		g.Add(func() error {
			klog.Infof("Starting metrics server: %s", metricsServerListenAddress)
			return web.ListenAndServe(&metricsServer, tlsConfig, promLogger)
		}, func(error) {
			ctxShutDown, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			if err := metricsServer.Shutdown(ctxShutDown); err != nil {
				klog.ErrorS(err, "shutdown metrics server")
			}
		})
	}

	if err := g.Run(); err != nil {
		klog.Fatalf("RunGroup Error: %v", err)
	}
	klog.Info("Exiting")
}

func buildTelemetryServer(registry prometheus.Gatherer) *http.ServeMux {
	mux := http.NewServeMux()

	// Add metricsPath
	mux.Handle(metricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{ErrorLog: promLogger{}}))
	// Add index
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(`<html>
             <head><title>Cluster-API-State-Metrics Metrics Server</title></head>
             <body>
             <h1>Cluster-API-State-Metrics Metrics</h1>
			 <ul>
             <li><a href='` + metricsPath + `'>metrics</a></li>
			 </ul>
             </body>
             </html>`)); err != nil {
			klog.ErrorS(err, "write telemetry index data")
		}
	})
	return mux
}

func buildMetricsServer(m http.Handler, durationObserver prometheus.ObserverVec) *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	mux.Handle(metricsPath, promhttp.InstrumentHandlerDuration(durationObserver, m))

	// Add healthzPath
	mux.HandleFunc(healthzPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(http.StatusText(http.StatusOK))); err != nil {
			klog.ErrorS(err, "write healthz text")
		}
	})
	// Add index
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(`<html>
             <head><title>Cluster-API Metrics Server</title></head>
             <body>
             <h1>Cluster-API Metrics</h1>
			 <ul>
             <li><a href='` + metricsPath + `'>metrics</a></li>
             <li><a href='` + healthzPath + `'>healthz</a></li>
			 </ul>
             </body>
             </html>`)); err != nil {
			klog.ErrorS(err, "write metrics index data")
		}
	})
	return mux
}

func getCtrlClient(kubeconfig, context string) (client.WithWatch, error) {
	// The kubeconfig flag in "sigs.k8s.io/controller-runtime/pkg/client/config"
	// is using another flagset. Because of that we use the value of the kubeconfig
	// flag directly and fallback to the config.GetConfigWithContext function.
	var cfg *rest.Config
	var err error
	if len(kubeconfig) > 0 {
		cfg, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
			&clientcmd.ConfigOverrides{
				ClusterInfo: clientcmdapi.Cluster{
					Server: "",
				},
				CurrentContext: context,
			}).ClientConfig()
	} else {
		cfg, err = config.GetConfigWithContext(os.Getenv("KUBECONTEXT"))
	}
	if err != nil {
		return nil, err
	}
	return client.NewWithWatch(cfg, client.Options{
		Scheme: scheme,
	})
}
