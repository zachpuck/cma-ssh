package apiserver

import (
	"context"

	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/common"
	"github.com/samsung-cnct/cma-ssh/pkg/generated/api"
	pb "github.com/samsung-cnct/cma-ssh/pkg/generated/api"
	"github.com/samsung-cnct/cma-ssh/pkg/ssh"
	"github.com/samsung-cnct/cma-ssh/pkg/util"

	corev1 "k8s.io/api/core/v1"
	clientlib "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func PrepareNodes(in *pb.CreateClusterMsg) ([]byte, []byte, error) {
	private, public, err := ssh.GenerateSSHKeyPair()
	if err != nil {
		return nil, nil, err
	}

	for _, node := range in.ControlPlaneNodes {
		sshParams := ssh.MachineParams{
			Username:   node.Username,
		}
		err := ssh.SetupPrivateKeyAccess(sshParams, private, public)
		if err != nil {
			return private, public, err
		}
	}

	for _, node := range in.WorkerNodes {
		sshParams := ssh.MachineParams{
			Username:   node.Username,
		}

		err := ssh.SetupPrivateKeyAccess(sshParams, private, public)
		if err != nil {
			return private, public, err
		}
	}

	return public, private, nil
}

func TranslateClusterStatus(crStatus common.ClusterStatusPhase) cmassh.ClusterStatus {

	var clusterStatus = cmassh.ClusterStatus_STATUS_UNSPECIFIED
	switch crStatus {
	case common.UnspecifiedClusterPhase:
		clusterStatus = cmassh.ClusterStatus_STATUS_UNSPECIFIED
	case common.ErrorClusterPhase:
		clusterStatus = cmassh.ClusterStatus_ERROR
	case common.RunningClusterPhase:
		clusterStatus = cmassh.ClusterStatus_RUNNING
	case common.StoppingClusterPhase:
		clusterStatus = cmassh.ClusterStatus_STOPPING
	case common.ReconcilingClusterPhase:
		clusterStatus = cmassh.ClusterStatus_RECONCILING

	}

	return clusterStatus
}

func GetKubeConfig(clusterName string, manager manager.Manager) ([]byte, error) {
	// get client
	client := manager.GetClient()

	// get kubeconfig secret
	kubeConfigSecret := &corev1.Secret{}
	err := client.Get(context.Background(), clientlib.ObjectKey{
		Namespace: clusterName,
		Name:      clusterName + "-kubeconfig",
	}, kubeConfigSecret)
	if err != nil {
		return nil, err
	}

	return kubeConfigSecret.Data[corev1.ServiceAccountKubeconfigKey], nil
}

func GetMachineName(clusterName string, hostIp string, manager manager.Manager) (string, error) {
	// get client
	client := manager.GetClient()

	// get list of cluster machines
	machineList, err := util.GetClusterMachineList(client, clusterName)
	if err != nil {
		return "", err
	}

	for _, machine := range machineList {
		if machine.Status.SshConfig.Host == hostIp {
			return machine.GetName(), nil
		}
	}

	return "", nil
}
