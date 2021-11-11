// SPDX-License-Identifier: MIT

package store

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
)

func TestClusterStore(t *testing.T) {
	startTime := 1501569018
	metav1StartTime := metav1.Unix(int64(startTime), 0)

	cases := []generateMetricsTestCase{
		{
			Obj: &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "cluster1",
					Namespace:         "ns1",
					CreationTimestamp: metav1StartTime,
					ResourceVersion:   "10596",
					UID:               types.UID("foo"),
				},
			},
			Want: `
				# HELP capi_cluster_created Unix creation timestamp
				# HELP capi_cluster_labels Kubernetes labels converted to Prometheus labels.
				# TYPE capi_cluster_created gauge
				# TYPE capi_cluster_labels gauge
				capi_cluster_created{cluster="cluster1",namespace="ns1",uid="foo"} 1.501569018e+09
				capi_cluster_labels{cluster="cluster1",namespace="ns1",uid="foo"} 1
				`,
			MetricNames: []string{"capi_cluster_labels", "capi_cluster_created"},
		},
		{
			Obj: &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "cluster2",
					Namespace:         "ns2",
					CreationTimestamp: metav1StartTime,
					ResourceVersion:   "10597",
					UID:               types.UID("foo"),
				},
				Status: clusterv1.ClusterStatus{
					Phase: string(clusterv1.ClusterPhaseFailed),
				},
			},
			Want: `
				# HELP capi_cluster_status_phase The clusters current phase.
				# TYPE capi_cluster_status_phase gauge
				capi_cluster_status_phase{cluster="cluster2",namespace="ns2",phase="Deleting",uid="foo"} 0
				capi_cluster_status_phase{cluster="cluster2",namespace="ns2",phase="Failed",uid="foo"} 1
				capi_cluster_status_phase{cluster="cluster2",namespace="ns2",phase="Pending",uid="foo"} 0
				capi_cluster_status_phase{cluster="cluster2",namespace="ns2",phase="Provisioned",uid="foo"} 0
				capi_cluster_status_phase{cluster="cluster2",namespace="ns2",phase="Provisioning",uid="foo"} 0
				capi_cluster_status_phase{cluster="cluster2",namespace="ns2",phase="Unknown",uid="foo"} 0
				`,
			MetricNames: []string{"capi_cluster_status_phase"},
		},
		{
			Obj: &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "cluster2",
					Namespace:         "ns2",
					CreationTimestamp: metav1StartTime,
					ResourceVersion:   "10597",
					UID:               types.UID("foo"),
				},
				Status: clusterv1.ClusterStatus{
					Phase: string(clusterv1.ClusterPhaseFailed),
					Conditions: clusterv1.Conditions{
						clusterv1.Condition{
							Type:   clusterv1.InfrastructureReadyCondition,
							Status: corev1.ConditionTrue,
						},
						clusterv1.Condition{
							Type:   clusterv1.ReadyCondition,
							Status: corev1.ConditionFalse,
						},
						clusterv1.Condition{
							Type:   clusterv1.ControlPlaneInitializedCondition,
							Status: corev1.ConditionUnknown,
						},
					},
				},
			},
			Want: `
				# HELP capi_cluster_status_condition The current status conditions of a cluster.
				# TYPE capi_cluster_status_condition gauge
				capi_cluster_status_condition{cluster="cluster2",condition="ControlPlaneInitialized",namespace="ns2",status="false",uid="foo"} 0
				capi_cluster_status_condition{cluster="cluster2",condition="ControlPlaneInitialized",namespace="ns2",status="true",uid="foo"} 0
				capi_cluster_status_condition{cluster="cluster2",condition="ControlPlaneInitialized",namespace="ns2",status="unknown",uid="foo"} 1
				capi_cluster_status_condition{cluster="cluster2",condition="InfrastructureReady",namespace="ns2",status="false",uid="foo"} 0
				capi_cluster_status_condition{cluster="cluster2",condition="InfrastructureReady",namespace="ns2",status="true",uid="foo"} 1
				capi_cluster_status_condition{cluster="cluster2",condition="InfrastructureReady",namespace="ns2",status="unknown",uid="foo"} 0
				capi_cluster_status_condition{cluster="cluster2",condition="Ready",namespace="ns2",status="false",uid="foo"} 1
				capi_cluster_status_condition{cluster="cluster2",condition="Ready",namespace="ns2",status="true",uid="foo"} 0
				capi_cluster_status_condition{cluster="cluster2",condition="Ready",namespace="ns2",status="unknown",uid="foo"} 0
			`,
			MetricNames: []string{"capi_cluster_status_condition"},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(clusterMetricFamilies(nil))
		c.Headers = generator.ExtractMetricFamilyHeaders(clusterMetricFamilies(nil))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
