package apiserver

import (
	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/common"
	v1alpha "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
	pb "github.com/samsung-cnct/cma-ssh/pkg/generated/api"
	"github.com/samsung-cnct/cma-ssh/pkg/util/k8sutil"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	clientlib "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func (s *Server) CreateCluster(ctx context.Context, in *pb.CreateClusterMsg) (*pb.CreateClusterReply, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("CreateCluster")

	cluster := TranslateCreateClusterMsg(in)
	if cluster.PrivateKey == "" {
		err := PrepareNodes(&cluster)
		if err != nil {
			log.Error(err, "Failed to prepare nodes")
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	// get client
	client := s.Manager.GetClient()

	// create namespace
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: cluster.Name,
		},
	}
	err := client.Create(ctx, namespace)
	if err != nil {
		log.Error(err, "Failed to create cluster namespace")
		return nil, status.Error(codes.Internal, err.Error())
	}

	// create secret
	dataMap := make(map[string][]byte)
	dataMap["private-key"] = []byte(cluster.PrivateKey)
	dataMap["pass-phrase"] = []byte("")
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-private-key",
			Namespace: cluster.Name,
		},
		Type: corev1.SecretTypeOpaque,
		Data: dataMap,
	}
	err = k8sutil.CreateSecret(client, secret)
	if err != nil {
		log.Error(err, "Failed to create cluster private key ")
		return nil, status.Error(codes.Internal, err.Error())
	}

	// create cluster
	clusterObject := &v1alpha.CnctCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Name,
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
		},
		Spec: v1alpha.ClusterSpec{
			ClusterNetwork: v1alpha.ClusterNetworkingConfig{
				Services: v1alpha.NetworkRanges{
					CIDRBlock: "10.96.0.0/12",
				},
				Pods: v1alpha.NetworkRanges{
					CIDRBlock: "10.244.0.0/16",
				},
				ServiceDomain: "cluster.local",
			},
			KubernetesVersion: cluster.K8SVersion,
		},
	}
	err = client.Create(ctx, clusterObject)
	if err != nil {
		log.Error(err, "Failed to create cluster object")
		return nil, status.Error(codes.Internal, err.Error())
	}

	// create control plane machines
	for _, machineConfig := range cluster.ControlPlaneNodes {
		machineObject := &v1alpha.CnctMachine{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "control-plane-",
				Namespace:    cluster.Name,
				Labels: map[string]string{
					"controller-tools.k8s.io": "1.0",
				},
			},
			Spec: v1alpha.MachineSpec{
				ClusterRef: cluster.Name,
				Roles:      []common.MachineRoles{common.MachineRoleMaster, common.MachineRoleEtcd},
				SshConfig: v1alpha.MachineSshConfigInfo{
					Username:   machineConfig.Username,
					Host:       machineConfig.Host,
					Port:       uint32(machineConfig.Port),
					PublicHost: machineConfig.PublicHost,
					Secret:     "cluster-private-key",
				},
			},
		}

		err = client.Create(ctx, machineObject)
		if err != nil {
			log.Error(err, "Failed to create control plane machine object")
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	// create worker plane machines
	for _, machineConfig := range cluster.ControlPlaneNodes {
		machineObject := &v1alpha.CnctMachine{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "worker-",
				Namespace:    cluster.Name,
				Labels: map[string]string{
					"controller-tools.k8s.io": "1.0",
				},
			},
			Spec: v1alpha.MachineSpec{
				ClusterRef: cluster.Name,
				Roles:      []common.MachineRoles{common.MachineRoleWorker},
				SshConfig: v1alpha.MachineSshConfigInfo{
					Username:   machineConfig.Username,
					Host:       machineConfig.Host,
					Port:       uint32(machineConfig.Port),
					PublicHost: machineConfig.PublicHost,
					Secret:     "cluster-private-key",
				},
			},
		}

		err = client.Create(ctx, machineObject)
		if err != nil {
			log.Error(err, "Failed to create worker machine object")
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &pb.CreateClusterReply{
		Ok: true,
		Cluster: &pb.ClusterItem{
			Id:     "stub",
			Name:   in.Name,
			Status: pb.ClusterStatus_PROVISIONING,
		},
	}, nil
}

func (s *Server) GetCluster(ctx context.Context, in *pb.GetClusterMsg) (*pb.GetClusterReply, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("GetCluster")
	// get client
	client := s.Manager.GetClient()

	// get kubeconfig secret
	kubeconfigBytes, err := GetKubeConfig(in.Name, s.Manager)
	if err != nil {
		log.Info("Could not get kubeconfig secret for cluster " + in.Name)
	}

	// get cluster
	clusterInstance := &v1alpha.CnctCluster{}
	err = client.Get(
		ctx,
		clientlib.ObjectKey{
			Namespace: in.Name,
			Name:      in.Name,
		}, clusterInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		log.Error(err, "Could not query for cluster "+in.Name)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.GetClusterReply{
		Ok: true,
		Cluster: &pb.ClusterDetailItem{
			Id:         "stub",
			Name:       in.Name,
			Status:     TranslateClusterStatus(clusterInstance.Status.Phase),
			Kubeconfig: string(kubeconfigBytes),
		},
	}, nil
}

func (s *Server) DeleteCluster(ctx context.Context, in *pb.DeleteClusterMsg) (*pb.DeleteClusterReply, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("DeleteCluster")

	// get client
	client := s.Manager.GetClient()

	// get cluster
	clusterInstance := &v1alpha.CnctCluster{}
	err := client.Get(
		ctx,
		clientlib.ObjectKey{
			Namespace: in.Name,
			Name:      in.Name,
		}, clusterInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		log.Error(err, "Could not query for cluster "+in.Name)
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = client.Delete(ctx, clusterInstance)
	if err != nil {
		log.Error(err, "Could not delete cluster cr"+in.Name)
		return nil, status.Error(codes.Internal, err.Error())
	}

	namespace := &corev1.Namespace{}
	err = client.Get(
		ctx,
		clientlib.ObjectKey{
			Namespace: "",
			Name:      in.Name,
		}, namespace)
	if err == nil {
		err = client.Delete(ctx, namespace)
		if err != nil {
			log.Error(err, "Could not delete namespace "+in.Name)
			return nil, status.Error(codes.Internal, err.Error())
		}
	}


	return &pb.DeleteClusterReply{Ok: true, Status: "Deleted"}, nil
}

func (s *Server) GetClusterList(ctx context.Context, in *pb.GetClusterListMsg) (reply *pb.GetClusterListReply, err error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("GetClusterList")

	// get client
	client := s.Manager.GetClient()

	// get list of cluster CRs
	clusterCrList := &v1alpha.CnctClusterList{}
	err = client.List(
		ctx,
		&clientlib.ListOptions{LabelSelector: labels.Everything()},
		clusterCrList)
	if err != nil {
		log.Error(err, "could not list cluster CRs")
		return &pb.GetClusterListReply{
			Ok: false,
		}, err
	}

	if len(clusterCrList.Items) == 0 {
		return &pb.GetClusterListReply{
			Ok:       true,
			Clusters: nil,
		}, nil
	}

	var clusters []*pb.ClusterItem
	for _, cluster := range clusterCrList.Items {
		clusterStatus := TranslateClusterStatus(cluster.Status.Phase)

		clusters = append(clusters, &pb.ClusterItem{
			Id:     "stub",
			Name:   cluster.GetName(),
			Status: clusterStatus,
		})
	}

	return &pb.GetClusterListReply{
		Ok:       true,
		Clusters: clusters,
	}, nil
}

func (s *Server) AdjustClusterNodes(ctx context.Context, in *pb.AdjustClusterMsg) (*pb.AdjustClusterReply, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("AdjustClusterNodes")

	// get client
	client := s.Manager.GetClient()

	// get cluster
	clusterInstance := &v1alpha.CnctCluster{}
	err := client.Get(
		ctx,
		clientlib.ObjectKey{
			Namespace: in.Name,
			Name:      in.Name,
		}, clusterInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		log.Error(err, "Could not query for cluster "+in.Name)
		return nil, status.Error(codes.Internal, err.Error())
	}

	for _, addedNode := range in.AddNodes {
		publicHost := addedNode.Publichost
		if publicHost == "" {
			publicHost = addedNode.Host
		}
		machineObject := &v1alpha.CnctMachine{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "worker-",
				Namespace:    clusterInstance.GetNamespace(),
				Labels: map[string]string{
					"controller-tools.k8s.io": "1.0",
				},
			},
			Spec: v1alpha.MachineSpec{
				ClusterRef: clusterInstance.GetName(),
				Roles:      []common.MachineRoles{common.MachineRoleWorker},
				SshConfig: v1alpha.MachineSshConfigInfo{
					Username:   addedNode.Username,
					Host:       addedNode.Host,
					Port:       uint32(addedNode.Port),
					PublicHost: publicHost,
					Secret:     "cluster-private-key",
				},
			},
		}

		err := client.Create(ctx, machineObject)
		if err != nil {
			log.Error(err, "Failed to create worker machine object")
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	for _, removedNode := range in.RemoveNodes {
		machineName, err := GetMachineName(clusterInstance.GetName(), removedNode.Host, s.Manager)
		if err != nil {
			log.Error(err, "Failed to get machine name")
			return nil, status.Error(codes.Internal, err.Error())
		}

		if machineName == "" {
			log.Error(err, "Got empty machine name")
			return nil, status.Error(codes.Internal, err.Error())
		}

		machineObject := &v1alpha.CnctMachine{}
		err = client.Get(ctx,
			clientlib.ObjectKey{
				Namespace: clusterInstance.GetNamespace(),
				Name:      machineName,
			}, machineObject)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, status.Error(codes.NotFound, err.Error())
			}
			log.Error(err, "Could not get machine "+machineName)
			return nil, status.Error(codes.Internal, err.Error())
		}

		err = client.Delete(ctx, machineObject)
		if err != nil {
			log.Error(err, "Could not delete machine "+machineName)
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &pb.AdjustClusterReply{Ok: true}, nil
}

func (s *Server) GetUpgradeClusterInformation(ctx context.Context, in *pb.GetUpgradeClusterInformationMsg) (*pb.GetUpgradeClusterInformationReply, error) {
	// TODO: Do not hard code this list.
	return &pb.GetUpgradeClusterInformationReply{
		Versions: []string{
			"1.10.6",
			"1.11.2",
		},
	}, nil
}

func (s *Server) UpgradeCluster(ctx context.Context, in *pb.UpgradeClusterMsg) (*pb.UpgradeClusterReply, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("UpgradeCluster")

	// get client
	client := s.Manager.GetClient()

	// get cluster
	clusterInstance := &v1alpha.CnctCluster{}
	err := client.Get(
		ctx,
		clientlib.ObjectKey{
			Namespace: in.Name,
			Name:      in.Name,
		}, clusterInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		log.Error(err, "Could not query for cluster "+in.Name)
		return nil, status.Error(codes.Internal, err.Error())
	}

	// update version
	clusterInstance.Spec.KubernetesVersion = in.Version
	err = client.Update(ctx, clusterInstance)
	if err != nil {
		log.Error(err, "Could update cluster "+in.Name)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.UpgradeClusterReply{
		Ok: true,
	}, nil
}
