package util

import (
	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/common"
	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
)

func GetStatus(machines *clusterv1alpha1.MachineList) common.ClusterStatusPhase {
	if len(machines.Items) == 0 {
		return common.UnspecifiedClusterPhase
	}

	if ContainsStatuses(machines, []common.MachineStatusPhase{common.ErrorMachinePhase}) {
		return common.ErrorClusterPhase
	}

	if ContainsStatuses(machines,
		[]common.MachineStatusPhase{
			common.DeletingMachinePhase,
			common.ProvisioningMachinePhase,
			common.UpgradingMachinePhase,
			"",
		}) {
		return common.ReconcilingClusterPhase
	}

	return common.RunningClusterPhase
}

func ContainsStatuses(machines *clusterv1alpha1.MachineList, ss []common.MachineStatusPhase) bool {
	for _, item := range machines.Items {
		for _, ssItem := range ss {
			if item.Status.Phase == ssItem {
				return true
			}
		}
	}
	return false
}

func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func RemoveString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
