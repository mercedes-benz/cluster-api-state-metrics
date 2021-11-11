// SPDX-License-Identifier: MIT

package options

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	koptions "k8s.io/kube-state-metrics/v2/pkg/options"
)

var (
	// DefaultNamespaces is the default namespace selector for selecting and filtering across all namespaces.
	DefaultNamespaces = koptions.NamespaceList{metav1.NamespaceAll}

	// DefaultResources represents the default set of resources in cluster-api-state-metrics.
	DefaultResources = koptions.ResourceSet{
		"clusters":             struct{}{},
		"machines":             struct{}{},
		"machinedeployments":   struct{}{},
		"kubeadmcontrolplanes": struct{}{},
		"machinesets":          struct{}{},
	}
)
