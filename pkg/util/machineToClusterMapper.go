package util

import (
	"context"
	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

// A mapper that returns all the Registries
type MachineToClusterMapper struct {
	client.Client
}

func (m MachineToClusterMapper) Map(obj handler.MapObject) []reconcile.Request {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("MachineToClusterMapper Map()")

	var res []reconcile.Request

	machine, ok := obj.Object.(*clusterv1alpha1.Machine)
	if !ok {
		return res // This wasn't a Machine
	}

	clusters := &clusterv1alpha1.ClusterList{}
	if err := m.List(context.Background(), &client.ListOptions{}, clusters); err != nil {
		log.Error(err, "could not get list of clusters")
		return res
	}

	// Add  the Cluster referred to by the machine
	for _, cluster := range clusters.Items {
		if cluster.GetName() == machine.Spec.ClusterRef {
			res = append(res, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      cluster.GetName(),
					Namespace: cluster.GetNamespace(),
				},
			})
		}
	}
	return res
}
