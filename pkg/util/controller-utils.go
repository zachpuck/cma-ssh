package util

import (
	"context"
	"fmt"
	"strings"

	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/common"
	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// kubernetesVersions are the versions we support installing.
// TODO: in future set this through ldflags in make
const kubernetesVersions = "1.12.6 1.13.4"

// KubernetesVersions() returns the list of all kubernetes versions cma-ssh supports.
func KubernetesVersions() []string {
	return strings.Split(kubernetesVersions, " ")
}

func GetClusterMachineList(c client.Client, clusterName string) ([]clusterv1alpha1.CnctMachine, error) {
	machineList := &clusterv1alpha1.CnctMachineList{}
	err := c.List(
		context.Background(),
		&client.ListOptions{LabelSelector: labels.Everything()},
		machineList)
	if err != nil {
		return nil, err
	}

	var clusterMachines []clusterv1alpha1.CnctMachine
	for _, item := range machineList.Items {
		if item.Spec.ClusterRef == clusterName {
			clusterMachines = append(clusterMachines, item)
		}
	}

	return clusterMachines, nil
}

// returns overall cluster status and api enpoint if available
func GetStatus(machines []clusterv1alpha1.CnctMachine) (common.ClusterStatusPhase, string) {

	if len(machines) == 0 {
		return common.UnspecifiedClusterPhase, ""
	}

	// if there is a Ready machine and it's a master, grab the api endpoint
	apiEndpoint := ""
	for _, machine := range machines {
		if machine.Status.Phase == common.ReadyMachinePhase && ContainsRole(machine.Spec.Roles, common.MachineRoleMaster) {
			apiEndpoint = machine.Status.SshConfig.Host + ":" + common.ApiEnpointPort
		}
	}

	if ContainsStatuses(machines, []common.MachineStatusPhase{common.ErrorMachinePhase}) {
		return common.ErrorClusterPhase, apiEndpoint
	}

	if ContainsStatuses(machines,
		[]common.MachineStatusPhase{
			common.DeletingMachinePhase,
			common.ProvisioningMachinePhase,
			common.UpgradingMachinePhase,
			"",
		}) {
		return common.ReconcilingClusterPhase, apiEndpoint
	}

	return common.RunningClusterPhase, apiEndpoint
}

// similar to GetStatus(), but returns true for whether its ok to proceed with machine deletion
// Deletion is ok when none of the machines in the cluster are in Creating or Upgrading state
func IsReadyForDeletion(machines []clusterv1alpha1.CnctMachine) bool {

	if len(machines) == 0 {
		return true
	}

	if ContainsStatuses(machines,
		[]common.MachineStatusPhase{
			common.ProvisioningMachinePhase,
			common.UpgradingMachinePhase,
			"",
		}) {
		return false
	}

	return true
}

// similar to GetStatus(), but returns true for whether its ok to proceed with machine upgrade
// Upgrade is ok when none of the machines in the cluster are in Creating state
// If some of the machines are in Error state, return an error
func IsReadyForUpgrade(machines []clusterv1alpha1.CnctMachine) (bool, error) {
	if len(machines) == 0 {
		return true, nil
	}

	if ContainsStatuses(machines,
		[]common.MachineStatusPhase{
			common.ProvisioningMachinePhase,
			"",
		}) {
		return false, nil
	}

	if ContainsStatuses(machines,
		[]common.MachineStatusPhase{
			common.ErrorMachinePhase,
		}) {
		return false, fmt.Errorf("cannot upgrade, some of the cluster machines are in error state")
	}

	return true, nil
}

func ContainsStatuses(machines []clusterv1alpha1.CnctMachine, ss []common.MachineStatusPhase) bool {
	for _, item := range machines {
		for _, ssItem := range ss {
			if item.Status.Phase == ssItem {
				return true
			}
		}
	}
	return false
}

func ContainsRole(slice []common.MachineRoles, s common.MachineRoles) bool {
	for _, item := range slice {
		if item == s {
			return true
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

func GetMaster(machines []clusterv1alpha1.CnctMachine) (*clusterv1alpha1.CnctMachine, error) {
	for _, item := range machines {
		for _, role := range item.Spec.Roles {
			if role == common.MachineRoleMaster {
				return &item, nil
			}
		}
	}

	return nil, fmt.Errorf("could not find master node")
}
