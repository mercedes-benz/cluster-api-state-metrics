<!-- SPDX-License-Identifier: MIT -->

[![CI](https://github.com/Daimler/cluster-api-state-metrics/actions/workflows/ci.yaml/badge.svg)](https://github.com/Daimler/cluster-api-state-metrics/actions/workflows/ci.yaml)
[![FOSS Scan](https://github.com/Daimler/cluster-api-state-metrics/actions/workflows/foss.yaml/badge.svg)](https://github.com/Daimler/cluster-api-state-metrics/actions/workflows/foss.yaml)

# Overview

cluster-api-state-metrics (CASM) is a service that listens to the Kubernetes API server and generates metrics about the state of custom resource objects related of [Kubernetes Cluster API].
This project is highly inspired by [kube-state-metrics] and shares some codebase with it and resources which are in scope for kube-state-metrics are not scope of cluster-api-state-metrics.

The metrics are exported on the HTTP endpoint `/metrics` via http (default port `8080`) and are served as plaintext.
The endpoint is designed to get consumed by Prometheus or a scraper which is compatible with a Prometheus client endpoint.
Kubernetes custom resource objects which get deleted are no longer visible to the `/metrics` endpoint.

[Kubernetes Cluster API]: https://cluster-api.sigs.k8s.io/
[kube-state-metrics]: https://github.com/kubernetes/kube-state-metrics

## Versioning

### Kubernetes Version

cluster-api-state-metrics uses the client of [`controller-runtime`] to talk with Kubernetes
clusters.
Because of that the supported Kubernetes cluster version is determined by `controller-runtime`.

[`controller-runtime`]: https://github.com/kubernetes-sigs/controller-runtime

### Kubernetes Cluster API Version

Resources of [Cluster API] can evolve, i.e. the group version for a resource may
change from alpha to beta and finally GA in a more recent Cluster API version.

Cluster API provides conversion webhooks for its custom resource definitions.
By that cluster-api-state-metrics may be compatible to different versions of Cluster API.

| casm \ capi | **v1alpha3** | **v1alpha4** | **v1beta1** |
|-------------|:------------:|:------------:|:-----------:|
| **v0.1.0**  |      -       |      ✓       |     (✓)     |
| **main**    |      -       |      ✓       |     (✓)     |

- `✓` Imported version used
- `(✓)` Version supported via conversion webhooks.
- `-` Not supported

[Cluster API]: https://github.com/kubernetes-sigs/cluster-api

# Usage Documentation

## CLI Arguments

[embedmd]:# (./help.txt)
```txt
cluster-api-state-metrics -h
Usage of ./bin/cluster-api-state-metrics:
      --add_dir_header                        If true, adds the file directory to the header of the log messages
      --alsologtostderr                       log to standard error as well as files
      --apiserver string                      The URL of the apiserver to use
      --enable-gzip-encoding                  Gzip responses when requested by clients via 'Accept-Encoding: gzip' header.
  -h, --help                                  Print Help text
      --host string                           Host to expose metrics on. (default "::")
      --kubeconfig string                     Absolute path to the kubeconfig file
      --log_backtrace_at traceLocation        when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                        If non-empty, write log files in this directory
      --log_file string                       If non-empty, use this log file
      --log_file_max_size uint                Defines the maximum size a log file can grow to. Unit is megabytes. If the value is 0, the maximum file size is unlimited. (default 1800)
      --logtostderr                           log to standard error instead of files (default true)
      --metric-allowlist string               Comma-separated list of metrics to be exposed. This list comprises of exact metric names and/or regex patterns. The allowlist and denylist are mutually exclusive.
      --metric-annotations-allowlist string   Comma-separated list of Kubernetes annotations keys that will be used in the resource' labels metric. By default the metric contains only name and namespace labels. To include additional annotations provide a list of resource names in their plural form and Kubernetes annotation keys you would like to allow for them (Example: '=namespaces=[kubernetes.io/team,...],pods=[kubernetes.io/team],...)'. A single '*' can be provided per resource instead to allow any annotations, but that has severe performance implications (Example: '=pods=[*]').
      --metric-denylist string                Comma-separated list of metrics not to be enabled. This list comprises of exact metric names and/or regex patterns. The allowlist and denylist are mutually exclusive.
      --metric-labels-allowlist string        Comma-separated list of additional Kubernetes label keys that will be used in the resource' labels metric. By default the metric contains only name and namespace labels. To include additional labels provide a list of resource names in their plural form and Kubernetes label keys you would like to allow for them (Example: '=namespaces=[k8s-label-1,k8s-label-n,...],pods=[app],...)'. A single '*' can be provided per resource instead to allow any labels, but that has severe performance implications (Example: '=pods=[*]').
      --namespaces string                     Comma-separated list of namespaces to be enabled. Defaults to ""
      --namespaces-denylist string            Comma-separated list of namespaces not to be enabled. If namespaces and namespaces-denylist are both set, only namespaces that are excluded in namespaces-denylist will be used.
      --one_output                            If true, only write logs to their native severity level (vs also writing to each lower severity level)
      --port int                              Port to expose metrics on. (default 8080)
      --resources string                      Comma-separated list of Resources to be enabled. Defaults to "clusters,kubeadmcontrolplanes,machinedeployments,machines,machinesets"
      --skip_headers                          If true, avoid header prefixes in the log messages
      --skip_log_headers                      If true, avoid headers when opening log files
      --stderrthreshold severity              logs at or above this threshold go to stderr (default 2)
      --telemetry-host string                 Host to expose kube-state-metrics self metrics on. (default "::")
      --telemetry-port int                    Port to expose kube-state-metrics self metrics on. (default 8081)
      --tls-config string                     Path to the TLS configuration file
      --use-apiserver-cache                   Sets resourceVersion=0 for ListWatch requests, using cached resources from the apiserver instead of an etcd quorum read.
  -v, --v Level                               number for the log level verbosity
      --version                               kube-state-metrics build version information
      --vmodule moduleSpec                    comma-separated list of pattern=N settings for file-filtered logging
```

### Building binary from source

To build cluster-api-state-metrics from the source code yourself you need to have a working Go environment with [version 1.17 or greater installed](https://golang.org/doc/install).

```shell
git clone https://github.com/daimler/cluster-api-state-metrics.git
cd cluster-api-state-metrics
make build
```

The Makefile provides several targets:

[embedmd]:# (./make-help.txt)
```txt

Usage:
  make <target>

General
  help             Display this help.

Development
  manifests        Generate WebhookConfiguration, ClusterRole objects.
  generate         Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
  fmt              Run go fmt against code.
  vet              Run go vet against code.
  spdxcheck        Run spdx check against all files.
  doccheck         Run docs specific checks
  test             Run tests.

Build
  docs             Regenerate docs
  build            Build manager binary.
  image            Build container image
  push             Push container image
  run              Run a controller from your host.

Deployment
  template         Create kustomized deployment yaml.
  deploy           Deploy controller to the K8s cluster specified in ~/.kube/config.
  undeploy         Undeploy controller from the K8s cluster specified in ~/.kube/config.
```

# Metrics Documentation

Metrics will be made available on port 8080 by default. Alternatively it is possible to pass the commandline flag `-addr` to override the port.
An overview of all metrics can be found in [metrics.md](docs/README.md).

# Contributing

We welcome any contributions.
If you want to contribute to this project, please read the [contributing guide](CONTRIBUTING.md).

# License

Full information on the license for this software is available in the [LICENSE](LICENSE) file.

Parts of the software are marked to be licensed under `Apache 2.0` which is due to this files got copied and modified from `kube-state-metrics`. Changes to this files are noted at the top of the file.

The latest artifacts of the `foss-scan` which includes the notices file for third party licenses and risk report are available at the artifacts section of the last [FOSS Scan workflow](https://github.com/Daimler/cluster-api-state-metrics/actions/workflows/foss.yaml?query=branch%3Amain).

# Provider Information

Please visit [https://www.daimler-tss.com/en/imprint/] for information on the provider.

Notice: Before you use the program in productive use, please take all necessary precautions, e.g. testing and verifying the program with regard to your specific use. The program was tested solely for our own use cases, which might differ from yours.

[https://www.daimler-tss.com/en/imprint/]: https://www.daimler-tss.com/en/imprint/
