// SPDX-License-Identifier: MIT

package store

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/kube-state-metrics/v2/pkg/customresource"
	"k8s.io/kube-state-metrics/v2/pkg/options"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha4"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var scheme = runtime.NewScheme()

func init() {
	_ = clusterv1.AddToScheme(scheme)
	_ = controlplanev1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

type ControllerRuntimeClientFactory struct{}

func Factories() []customresource.RegistryFactory {
	return []customresource.RegistryFactory{
		&ClusterFactory{},
		&KubeadmControlPlaneFactory{},
		&MachineDeploymentFactory{},
		&MachineSetFactory{},
		&MachineFactory{},
	}
}

func (f *ControllerRuntimeClientFactory) CreateClient(cfg *rest.Config) (interface{}, error) {
	return client.NewWithWatch(cfg, client.Options{
		Scheme: scheme,
	})
}

var DefaultResources = options.ResourceSet{}

func init() {
	if len(DefaultResources) == 0 {
		for _, factory := range Factories() {
			DefaultResources[factory.Name()] = struct{}{}
		}
	}
}
