package util

import (
	"context"

	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// A mapper that returns all the Registries
type MachineToClusterMapper struct {
	client.Client
}

func (m MachineToClusterMapper) Map(obj handler.MapObject) []reconcile.Request {
	var res []reconcile.Request

	machine, ok := obj.Object.(*clusterv1alpha1.CnctMachine)
	if !ok {
		return res // This wasn't a Machine
	}

	clusters := &clusterv1alpha1.CnctClusterList{}
	if err := m.List(context.Background(), &client.ListOptions{}, clusters); err != nil {
		klog.Errorf("could not get list of clusters: %q", err)
		return res
	}

	// Add  the Cluster referred to by the machine
	for _, cluster := range clusters.Items {
		clusterName := GetClusterNameFromMachineOwnerRef(machine)
		if cluster.GetName() == clusterName {
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
