// SPDX-License-Identifier: MIT

package store

import (
	"k8s.io/client-go/tools/cache"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha4"
)

var availableStores = map[string]func(f *Builder) []cache.Store{
	"clusters":             func(b *Builder) []cache.Store { return b.buildClusterStore() },
	"machines":             func(b *Builder) []cache.Store { return b.buildMachineStore() },
	"machinedeployments":   func(b *Builder) []cache.Store { return b.buildMachineDeploymentStore() },
	"kubeadmcontrolplanes": func(b *Builder) []cache.Store { return b.buildKubeadmControlPlaneStore() },
	"machinesets":          func(b *Builder) []cache.Store { return b.buildMachineSetStore() },
}

func (b *Builder) buildClusterStore() []cache.Store {
	return b.buildStoresFunc(clusterMetricFamilies(b.allowLabelsList["clusters"]), &clusterv1.Cluster{}, createClusterListWatch, b.useAPIServerCache)
}

func (b *Builder) buildMachineStore() []cache.Store {
	return b.buildStoresFunc(machineMetricFamilies(b.allowLabelsList["machine"]), &clusterv1.Machine{}, createMachineListWatch, b.useAPIServerCache)
}

func (b *Builder) buildMachineDeploymentStore() []cache.Store {
	return b.buildStoresFunc(machineDeploymentMetricFamilies(b.allowLabelsList["machinedeployment"]), &clusterv1.MachineDeployment{}, createMachineDeploymentListWatch, b.useAPIServerCache)
}

func (b *Builder) buildKubeadmControlPlaneStore() []cache.Store {
	return b.buildStoresFunc(kubeadmControlPlaneMetricFamilies(b.allowLabelsList["kubeadmcontrolplane"]), &controlplanev1.KubeadmControlPlane{}, createKubeadmControlPlaneListWatch, b.useAPIServerCache)
}

func (b *Builder) buildMachineSetStore() []cache.Store {
	return b.buildStoresFunc(machineSetMetricFamilies(b.allowLabelsList["machineset"]), &clusterv1.MachineSet{}, createMachineSetListWatch, b.useAPIServerCache)
}
