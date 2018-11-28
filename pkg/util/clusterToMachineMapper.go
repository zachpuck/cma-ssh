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
type ClusterToMachineMapper struct {
	client.Client
}

func (m ClusterToMachineMapper) Map(obj handler.MapObject) []reconcile.Request {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("clusterToMachineMapper Map()")

	var res []reconcile.Request

	cluster, ok := obj.Object.(*clusterv1alpha1.Cluster)
	if !ok {
		return res // This wasn't a Cluster
	}

	machines := &clusterv1alpha1.MachineList{}
	if err := m.List(context.Background(), &client.ListOptions{}, machines); err != nil {
		log.Error(err, "could not get list of machines")
		return res
	}

	// Add all the Machines that are members of this Cluster
	for _, machine := range machines.Items {
		if machine.Spec.ClusterRef == cluster.GetName() {
			res = append(res, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      machine.GetName(),
					Namespace: machine.GetNamespace(),
				},
			})
		}
	}
	return res
}
