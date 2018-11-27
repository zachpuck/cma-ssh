/*
Copyright 2018 Samsung SDS.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cluster

import (
	"context"
	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/common"
	"github.com/samsung-cnct/cma-ssh/pkg/util"
	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new Cluster Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCluster{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("cluster Controller Add()")

	// Create a new controller
	c, err := controller.New("cluster-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Cluster
	err = c.Watch(&source.Kind{Type: &clusterv1alpha1.Cluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch Machine objects with corresponding cluster name
	err = c.Watch(
		&source.Kind{Type: &clusterv1alpha1.Machine{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {

				machineList := &clusterv1alpha1.MachineList{}
				err = mgr.GetClient().List(
					context.TODO(),
					&client.ListOptions{LabelSelector: labels.Everything()},
					machineList)
				if err != nil {
					log.Error(err, "could not list Machines")
					return []reconcile.Request{}
				}

				var keys []reconcile.Request
				for _, machineInstance := range machineList.Items {
					if machineInstance.Spec.ClusterName == a.Meta.GetName() {
						keys = append(keys, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Namespace: machineInstance.GetNamespace(),
								Name:      machineInstance.GetName(),
							},
						})
					}
				}

				// return found keys
				return keys
			}),
		}, predicate.ResourceVersionChangedPredicate{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileCluster{}

// ReconcileCluster reconciles a Cluster object
type ReconcileCluster struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Cluster object and makes changes based on the state read
// and what is in the Cluster.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=cluster.sds.samsung.com,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=machine.sds.samsung.com,resources=clusters,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("cluster Controller Reconcile()")

	// Fetch the Cluster instance
	clusterInstance := &clusterv1alpha1.Cluster{}
	err := r.Get(context.TODO(), request.NamespacedName, clusterInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	machineList := &clusterv1alpha1.MachineList{}
	err = r.Client.List(
		context.TODO(),
		&client.ListOptions{LabelSelector: labels.Everything()},
		machineList)
	if err != nil {
		log.Error(err, "could not list Machines")
		return reconcile.Result{}, err
	}

	var machineStatuses []common.StatusPhase
	for _, machineInstance := range machineList.Items {
		if machineInstance.Spec.ClusterName == clusterInstance.GetName() {
			machineStatuses = append(machineStatuses, machineInstance.Status.Phase)
		}
	}

	clusterStatus, err := util.GetStatus(machineStatuses)
	if err != nil {
		log.Error(err, "could not get cluster status")
		return reconcile.Result{}, err
	}

	// TODO: check whether current machine statuses are either
	// 1) Some 'ready', some uniformly something else - set the current status to the other non ready status
	// 2) Some 'ready', some are non-uniformly something else (this is an error)
	// 3) No machines - set status to EmptyClusterPhase
	// 4) All ready - set status to ReadyResourcePhase

	// TODO: update status

	return reconcile.Result{}, nil
}
