# cluster-api-state-metrics

This helm chart allows you to deploy `cluster-api-state-metrics` to a Kubernetes cluster.

## Installation

In order to install the chart, use the helm command below:
```bash
# TODO: specify helm repo to use for publishing the helm chart
helm repo add ${HELM_REPO} ${HELM_REPO_URL}
helm install cluster-api-state-metrics ${HELM_REPO}/cluster-api-state-metrics
```

## Values
The only documented values are the ones that were added in addition to the default ones provided by Helm.

Most of the values documented below are simply passed to the `cluster-api-state-metrics` app as arguments(see [CLI Arguments](https://github.com/mercedes-benz/cluster-api-state-metrics#cli-arguments))

| variable | Default value | Description |
| -------- | ----- | ----- |
| `config.addDirHeader` | `false` | If true, adds the file directory to the header of the log messages |
| `config.alsoLogtoStderr` | `false` | Log to standard error as well as files |
| `config.enableGzipEncoding` | `false` | Gzip responses when requested by clients via 'Accept-Encoding: gzip' header. |
| `config.logBacktraceAt` | `""` | when logging hits line file:N (eg: main.go:50), emit a stack trace. |
| `config.logDir` | `""` | If non-empty, write log files in this directory |
| `config.logFile` | `""` | If non-empty, use this log file |
| `config.logFileMaxSize` | `1800` | Defines the maximum size a log file can grow to. Unit is megabytes. If the value is 0, the maximum file size is unlimited. (default 1800) |
| `config.logLevel` | `1` | number for the log level verbosity |  
| `config.logToStderr` | `true` | log to standard error instead of files (default true) |
| `config.metricAllowlist` | `""` | Comma-separated list of metrics to be exposed. This list comprises of exact metric names and/or regex patterns. The allowlist and denylist are mutually exclusive. |  
| `config.metricAnnotationsAllowlist` | `""` | Comma-separated list of Kubernetes annotations keys that will be used in the resource' labels metric. By default the metric contains only name and namespace labels. To include additional annotations provide a list of resource names in their plural form and Kubernetes annotation keys you would like to allow for them (Example: '=namespaces=[kubernetes.io/team,...],pods=[kubernetes.io/team],...)'. A single '*' can be provided per resource instead to allow any annotations, but that has severe performance implications (Example: '=pods=[*]'). |  
| `config.metricDenylist` | `""` | Comma-separated list of metrics not to be enabled. This list comprises of exact metric names and/or regex patterns. The allowlist and denylist are mutually exclusive. |  
| `config.metricLabelsAllowlist` | `""` | Comma-separated list of additional Kubernetes label keys that will be used in the resource' labels metric. By default the metric contains only name and namespace labels. To include additional labels provide a list of resource names in their plural form and Kubernetes label keys you would like to allow for them (Example: '=namespaces=[k8s-label-1,k8s-label-n,...],pods=[app],...)'. A single '*' can be provided per resource instead to allow any labels, but that has severe performance implications (Example: '=pods=[*]'). |
| `config.namespaces` | `""` | Comma-separated list of namespaces to be enabled. Defaults to "" which means all | 
| `config.namespacesDenylist` | `""` | Comma-separated list of namespaces not to be enabled. If namespaces and namespaces-denylist are both set, only namespaces that are excluded in namespaces-denylist will be used. |   
| `config.oneOutput` | `false` | If true, only write logs to their native severity level (vs also writing to each lower severity level) |  
| `config.port` | `8080` | Port to expose metrics on. (default 8080) |  
| `config.resources` | `"clusters,kubeadmcontrolplanes,machinedeployments,machines,machinesets"` | Comma-separated list of Resources to be enabled. |
| `config.skipHeaders` | `false` | If true, avoid header prefixes in the log messages |
| `config.skipLogHeaders` | `false` | If true, avoid headers when opening log files |
| `config.stderrThreshold` | `2` | logs at or above this threshold go to stderr (default 2) |
| `config.telemetryPort` | `8081` | Port to expose kube-state-metrics self metrics on. (default 8081) |  
| `config.useApiserverCache` | `false` | Sets resourceVersion=0 for ListWatch requests, using cached resources from the apiserver instead of an etcd quorum read. |
| `prometheusServiceMonitor.create` | `true` |  |
| `prometheusServiceMonitor.serviceMonitorSelectorLabels` | `{}` | Set the labels here if using serviceMonitorSelector. See https://prometheus-operator.dev/docs/operator/api/#prometheusspec |
| `prometheusServiceMonitor.capiMetrics.relabelings` | `{}` | Relabeling config used for the CAPI metrics (For an example, check [values.yaml](./cluster-api-state-metrics/values.yaml)) |
| `prometheusServiceMonitor.capiMetrics.metricRelabelings` | `{}` | Metric relabeling config used for the CAPI metrics |
| `prometheusServiceMonitor.exporterMetrics.relabelings` | `{}` | Relabeling config used for the CAPI exporter self metrics |
| `prometheusServiceMonitor.exporterMetrics.metricRelabelings` | `{}` | Metric relabeling config used for the CAPI exporter self metrics |
| `grafanaDashboards.create` | `false` | If true, create grafanaDashboard. This requires the `grafanadashboards.integreatly.org` CRD provided by the [grafana-operator](https://github.com/grafana-operator/grafana-operator) to be installed  |
