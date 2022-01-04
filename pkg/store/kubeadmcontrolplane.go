// SPDX-License-Identifier: MIT

package store

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha4"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=kubeadmcontrolplanes,verbs=get;list;watch

var descKubeadmControlPlaneLabelsDefaultLabels = []string{"namespace", "kubeadmcontrolplane", "uid"}

type KubeadmControlPlaneFactory struct {
	*ControllerRuntimeClientFactory
}

func (f *KubeadmControlPlaneFactory) Name() string {
	return "kubeadmcontrolplanes"
}

func (f *KubeadmControlPlaneFactory) ExpectedType() interface{} {
	return &controlplanev1.KubeadmControlPlane{}
}

func (f *KubeadmControlPlaneFactory) MetricFamilyGenerators(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGenerator(
			"capi_kubeadmcontrolplane_labels",
			"Kubernetes labels converted to Prometheus labels.",
			metric.Gauge,
			"",
			wrapKubeadmControlPlaneFunc(func(kcp *controlplanev1.KubeadmControlPlane) *metric.Family {
				labelKeys, labelValues := createLabelKeysValues(kcp.Labels, allowLabelsList)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"capi_kubeadmcontrolplane_created",
			"Unix creation timestamp",
			metric.Gauge,
			"",
			wrapKubeadmControlPlaneFunc(func(kcp *controlplanev1.KubeadmControlPlane) *metric.Family {
				ms := []*metric.Metric{}

				if !kcp.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(kcp.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),

		*generator.NewFamilyGenerator(
			"capi_kubeadmcontrolplane_status_replicas",
			"The number of replicas per kubeadmcontrolplane.",
			metric.Gauge,
			"",
			wrapKubeadmControlPlaneFunc(func(kcp *controlplanev1.KubeadmControlPlane) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(kcp.Status.Replicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"capi_kubeadmcontrolplane_status_replicas_ready",
			"The number of ready replicas per kubeadmcontrolplane.",
			metric.Gauge,
			"",
			wrapKubeadmControlPlaneFunc(func(kcp *controlplanev1.KubeadmControlPlane) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(kcp.Status.ReadyReplicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"capi_kubeadmcontrolplane_status_replicas_unavailable",
			"The number of unavailable replicas per kubeadmcontrolplane.",
			metric.Gauge,
			"",
			wrapKubeadmControlPlaneFunc(func(kcp *controlplanev1.KubeadmControlPlane) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(kcp.Status.UnavailableReplicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"capi_kubeadmcontrolplane_status_replicas_updated",
			"The number of updated replicas per kubeadmcontrolplane.",
			metric.Gauge,
			"",
			wrapKubeadmControlPlaneFunc(func(kcp *controlplanev1.KubeadmControlPlane) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(kcp.Status.UpdatedReplicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"capi_kubeadmcontrolplane_spec_replicas",
			"Number of desired replicas for a kubeadmcontrolplane.",
			metric.Gauge,
			"",
			wrapKubeadmControlPlaneFunc(func(kcp *controlplanev1.KubeadmControlPlane) *metric.Family {
				ms := []*metric.Metric{}

				if kcp.Spec.Replicas != nil {
					ms = append(ms, &metric.Metric{
						Value: float64(*kcp.Spec.Replicas),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"capi_kubeadmcontrolplane_spec_strategy_rollingupdate_max_surge",
			"Maximum number of replicas that can be scheduled above the desired number of replicas during a rolling update of a kubeadmcontrolplane.",
			metric.Gauge,
			"",
			wrapKubeadmControlPlaneFunc(func(kcp *controlplanev1.KubeadmControlPlane) *metric.Family {
				if kcp.Spec.RolloutStrategy == nil || kcp.Spec.RolloutStrategy.RollingUpdate == nil || kcp.Spec.Replicas == nil {
					return &metric.Family{}
				}

				maxSurge, err := intstr.GetScaledValueFromIntOrPercent(kcp.Spec.RolloutStrategy.RollingUpdate.MaxSurge, int(*kcp.Spec.Replicas), true)
				if err != nil {
					panic(err)
				}

				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(maxSurge),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"capi_kubeadmcontrolplane_owner",
			"Information about the kubeadmcontrolplane's owner.",
			metric.Gauge,
			"",
			wrapKubeadmControlPlaneFunc(func(kcp *controlplanev1.KubeadmControlPlane) *metric.Family {
				return getOwnerMetric(kcp.GetOwnerReferences())
			}),
		),
		*generator.NewFamilyGenerator(
			"capi_kubeadmcontrolplane_info",
			"Information about a kubeadmcontrolplane.",
			metric.Gauge,
			"",
			wrapKubeadmControlPlaneFunc(func(kcp *controlplanev1.KubeadmControlPlane) *metric.Family {
				labelKeys := []string{
					"version",
				}
				labelValues := []string{
					kcp.Spec.Version,
				}

				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
							Value:       1,
						},
					},
				}
			}),
		),
	}
}

func (f *KubeadmControlPlaneFactory) ListWatch(customResourceClient interface{}, ns string, fieldSelector string) cache.ListerWatcher {
	ctrlClient := customResourceClient.(client.WithWatch)
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			kubeadmControlPlaneList := controlplanev1.KubeadmControlPlaneList{}
			opts.FieldSelector = fieldSelector
			err := ctrlClient.List(context.TODO(), &kubeadmControlPlaneList, &client.ListOptions{Raw: &opts, Namespace: ns})
			return &kubeadmControlPlaneList, err
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			kubeadmControlPlaneList := controlplanev1.KubeadmControlPlaneList{}
			opts.FieldSelector = fieldSelector
			return ctrlClient.Watch(context.TODO(), &kubeadmControlPlaneList, &client.ListOptions{Raw: &opts, Namespace: ns})
		},
	}
}

func wrapKubeadmControlPlaneFunc(f func(*controlplanev1.KubeadmControlPlane) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		kubeadmControlPlane := obj.(*controlplanev1.KubeadmControlPlane)

		metricFamily := f(kubeadmControlPlane)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descKubeadmControlPlaneLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{kubeadmControlPlane.Namespace, kubeadmControlPlane.Name, string(kubeadmControlPlane.UID)}, m.LabelValues...)
		}

		return metricFamily
	}
}
