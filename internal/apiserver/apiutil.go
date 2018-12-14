package apiserver

import (
	"context"
	"encoding/base64"
	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/common"
	"github.com/samsung-cnct/cma-ssh/pkg/generated/api"
	pb "github.com/samsung-cnct/cma-ssh/pkg/generated/api"
	"github.com/samsung-cnct/cma-ssh/pkg/ssh"
	"github.com/samsung-cnct/cma-ssh/pkg/util"
	corev1 "k8s.io/api/core/v1"
	clientlib "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TranslateCreateClusterMsg(in *pb.CreateClusterMsg) ssh.SSHClusterParams {
	cluster := ssh.SSHClusterParams{
		Name:       in.Name,
		K8SVersion: in.K8SVersion,
		PrivateKey: in.PrivateKey,
	}

	for _, m := range in.ControlPlaneNodes {
		cluster.ControlPlaneNodes = append(cluster.ControlPlaneNodes, ssh.SSHMachineParams{
			Username: m.Username,
			Password: m.Password,
			Host:     m.Host,
			Port:     m.Port,
		})
	}

	for _, m := range in.WorkerNodes {
		cluster.WorkerNodes = append(cluster.WorkerNodes, ssh.SSHMachineParams{
			Username: m.Username,
			Password: m.Password,
			Host:     m.Host,
			Port:     m.Port,
		})
	}

	return cluster
}

func PrepareNodes(cluster *ssh.SSHClusterParams) error {
	private, public, err := ssh.GenerateSSHKeyPair()
	if err != nil {
		return err
	}

	cluster.PrivateKey = base64.StdEncoding.EncodeToString(private)
	cluster.PublicKey = base64.StdEncoding.EncodeToString(public)

	for _, node := range cluster.ControlPlaneNodes {
		err := ssh.SetupPrivateKeyAccess(node, private, public)
		if err != nil {
			return err
		}
	}

	for _, node := range cluster.WorkerNodes {
		err := ssh.SetupPrivateKeyAccess(node, private, public)
		if err != nil {
			return err
		}
	}

	return nil
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
		if machine.Spec.SshConfig.Host == hostIp {
			return machine.GetName(), nil
		}
	}

	return "", nil
}
