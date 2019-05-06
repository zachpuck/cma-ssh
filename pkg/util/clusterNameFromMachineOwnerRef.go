package util

import (
	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
)

const (
	ClusterKind = "CnctCluster"
)

// GetClusterNameFromMachineOwnerRef gets the cluster name from the machine owner reference
func GetClusterNameFromMachineOwnerRef(machineInstance *clusterv1alpha1.CnctMachine) string {
	for i := range machineInstance.OwnerReferences {
		if machineInstance.OwnerReferences[i].Kind == ClusterKind {
			return machineInstance.OwnerReferences[i].Name
		}
	}

	return ""
}
