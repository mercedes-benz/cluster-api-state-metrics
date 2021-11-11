/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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
[1]: https://github.com/kubernetes/kube-state-metrics/blob/e859b280fcc2/pkg/options/options.go

The original source was adjusted to:
- re-use types of the original package.
- remove sharding functionality to reduce complexity for the initial implementation.
*/

package options

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
	koptions "k8s.io/kube-state-metrics/v2/pkg/options"
)

// Options are the configurable parameters for kube-state-metrics.
type Options struct {
	Apiserver            string
	Kubeconfig           string
	Help                 bool
	Port                 int
	Host                 string
	TelemetryPort        int
	TelemetryHost        string
	TLSConfig            string
	Resources            koptions.ResourceSet
	Namespaces           koptions.NamespaceList
	NamespacesDenylist   koptions.NamespaceList
	MetricDenylist       koptions.MetricSet
	MetricAllowlist      koptions.MetricSet
	Version              bool
	AnnotationsAllowList koptions.LabelsAllowList
	LabelsAllowList      koptions.LabelsAllowList

	EnableGZIPEncoding bool

	UseAPIServerCache bool

	flags *pflag.FlagSet
}

// NewOptions returns a new instance of `Options`.
func NewOptions() *Options {
	return &Options{
		Resources:       koptions.ResourceSet{},
		MetricAllowlist: koptions.MetricSet{},
		MetricDenylist:  koptions.MetricSet{},
		LabelsAllowList: koptions.LabelsAllowList{},
	}
}

// AddFlags populated the Options struct from the command line arguments passed.
func (o *Options) AddFlags() {
	o.flags = pflag.NewFlagSet("", pflag.ExitOnError)
	// add klog flags
	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)
	o.flags.AddGoFlagSet(klogFlags)
	if err := o.flags.Lookup("logtostderr").Value.Set("true"); err != nil {
		klog.Fatal("error setting logtostderr default value")
	}
	o.flags.Lookup("logtostderr").DefValue = "true"
	o.flags.Lookup("logtostderr").NoOptDefVal = "true"

	o.flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		o.flags.PrintDefaults()
	}

	o.flags.BoolVarP(&o.UseAPIServerCache, "use-apiserver-cache", "", false, "Sets resourceVersion=0 for ListWatch requests, using cached resources from the apiserver instead of an etcd quorum read.")
	o.flags.StringVar(&o.Apiserver, "apiserver", "", `The URL of the apiserver to use`)
	o.flags.StringVar(&o.Kubeconfig, "kubeconfig", "", "Absolute path to the kubeconfig file")
	o.flags.StringVar(&o.TLSConfig, "tls-config", "", "Path to the TLS configuration file")
	o.flags.BoolVarP(&o.Help, "help", "h", false, "Print Help text")
	o.flags.IntVar(&o.Port, "port", 8080, `Port to expose metrics on.`)
	o.flags.StringVar(&o.Host, "host", "::", `Host to expose metrics on.`)
	o.flags.IntVar(&o.TelemetryPort, "telemetry-port", 8081, `Port to expose kube-state-metrics self metrics on.`)
	o.flags.StringVar(&o.TelemetryHost, "telemetry-host", "::", `Host to expose kube-state-metrics self metrics on.`)
	o.flags.Var(&o.Resources, "resources", fmt.Sprintf("Comma-separated list of Resources to be enabled. Defaults to %q", &DefaultResources))
	o.flags.Var(&o.Namespaces, "namespaces", fmt.Sprintf("Comma-separated list of namespaces to be enabled. Defaults to %q", &DefaultNamespaces))
	o.flags.Var(&o.NamespacesDenylist, "namespaces-denylist", "Comma-separated list of namespaces not to be enabled. If namespaces and namespaces-denylist are both set, only namespaces that are excluded in namespaces-denylist will be used.")
	o.flags.Var(&o.MetricAllowlist, "metric-allowlist", "Comma-separated list of metrics to be exposed. This list comprises of exact metric names and/or regex patterns. The allowlist and denylist are mutually exclusive.")
	o.flags.Var(&o.MetricDenylist, "metric-denylist", "Comma-separated list of metrics not to be enabled. This list comprises of exact metric names and/or regex patterns. The allowlist and denylist are mutually exclusive.")
	o.flags.Var(&o.AnnotationsAllowList, "metric-annotations-allowlist", "Comma-separated list of Kubernetes annotations keys that will be used in the resource' labels metric. By default the metric contains only name and namespace labels. To include additional annotations provide a list of resource names in their plural form and Kubernetes annotation keys you would like to allow for them (Example: '=namespaces=[kubernetes.io/team,...],pods=[kubernetes.io/team],...)'. A single '*' can be provided per resource instead to allow any annotations, but that has severe performance implications (Example: '=pods=[*]').")
	o.flags.Var(&o.LabelsAllowList, "metric-labels-allowlist", "Comma-separated list of additional Kubernetes label keys that will be used in the resource' labels metric. By default the metric contains only name and namespace labels. To include additional labels provide a list of resource names in their plural form and Kubernetes label keys you would like to allow for them (Example: '=namespaces=[k8s-label-1,k8s-label-n,...],pods=[app],...)'. A single '*' can be provided per resource instead to allow any labels, but that has severe performance implications (Example: '=pods=[*]').")

	o.flags.BoolVarP(&o.Version, "version", "", false, "kube-state-metrics build version information")
	o.flags.BoolVar(&o.EnableGZIPEncoding, "enable-gzip-encoding", false, "Gzip responses when requested by clients via 'Accept-Encoding: gzip' header.")
}

// Parse parses the flag definitions from the argument list.
func (o *Options) Parse() error {
	err := o.flags.Parse(os.Args)
	return err
}

// Usage is the function called when an error occurs while parsing flags.
func (o *Options) Usage() {
	o.flags.Usage()
}
