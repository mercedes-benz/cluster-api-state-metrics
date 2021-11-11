// SPDX-License-Identifier: MIT

package store

import (
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kube-state-metrics/v2/pkg/metric"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
)

func getConditionMetricFamily(conditions clusterv1.Conditions) *metric.Family {
	ms := make([]*metric.Metric, len(conditions)*len(conditionStatuses))

	for i, c := range conditions {
		conditionMetrics := addConditionMetrics(c.Status)

		for j, m := range conditionMetrics {
			selectedMetric := m

			selectedMetric.LabelKeys = []string{"condition", "status"}
			selectedMetric.LabelValues = append([]string{string(c.Type)}, selectedMetric.LabelValues...)
			ms[i*len(conditionStatuses)+j] = selectedMetric
		}
	}

	return &metric.Family{
		Metrics: ms,
	}
}

func getOwnerMetric(owners []metav1.OwnerReference) *metric.Family {
	if len(owners) == 0 {
		return &metric.Family{
			Metrics: []*metric.Metric{
				{
					LabelKeys:   []string{"owner_kind", "owner_name", "owner_is_controller"},
					LabelValues: []string{"<none>", "<none>", "<none>"},
					Value:       1,
				},
			},
		}
	}

	ms := make([]*metric.Metric, len(owners))

	for i, owner := range owners {
		if owner.Controller != nil {
			ms[i] = &metric.Metric{
				LabelValues: []string{owner.Kind, owner.Name, strconv.FormatBool(*owner.Controller)},
			}
		} else {
			ms[i] = &metric.Metric{
				LabelValues: []string{owner.Kind, owner.Name, "false"},
			}
		}
	}

	for _, m := range ms {
		m.LabelKeys = []string{"owner_kind", "owner_name", "owner_is_controller"}
		m.Value = 1
	}

	return &metric.Family{
		Metrics: ms,
	}
}
