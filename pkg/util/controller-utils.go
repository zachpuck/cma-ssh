package util

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/common"
	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
)

// returns overall cluster status and api enpoint if available
func GetStatus(machines []clusterv1alpha1.Machine) (common.ClusterStatusPhase, string) {

	if len(machines) == 0 {
		return common.UnspecifiedClusterPhase, ""
	}

	// if there is a Ready machine and it's a master, grab the api endpoint
	apiEndpoint := ""
	for _, machine := range machines {
		if machine.Status.Phase == common.ReadyMachinePhase && ContainsRole(machine.Spec.Roles, common.MachineRoleMaster) {
			apiEndpoint = machine.Spec.SshConfig.Host + ":" + common.ApiEnpointPort
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

func ContainsStatuses(machines []clusterv1alpha1.Machine, ss []common.MachineStatusPhase) bool {
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


const validBootstrapTokenChars = "0123456789abcdefghijklmnopqrstuvwxyz"
const BootstrapTokenIDBytes = 6
const BootstrapTokenSecretBytes = 16

func GenerateBootstrapToken() (string, error) {
	tokenID, err := randBytes(BootstrapTokenIDBytes)
	if err != nil {
		return "", err
	}

	tokenSecret, err := randBytes(BootstrapTokenSecretBytes)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s.%s", tokenID, tokenSecret), nil
}

func randBytes(length int) (string, error) {
	const maxByteValue = 252

	var (
		b     byte
		err   error
		token = make([]byte, length)
	)

	reader := bufio.NewReaderSize(rand.Reader, length*2)
	for i := range token {
		for {
			if b, err = reader.ReadByte(); err != nil {
				return "", err
			}
			if b < maxByteValue {
				break
			}
		}

		token[i] = validBootstrapTokenChars[int(b)%len(validBootstrapTokenChars)]
	}

	return string(token), nil
}

