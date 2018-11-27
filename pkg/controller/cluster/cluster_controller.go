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
	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
	"github.com/samsung-cnct/cma-ssh/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/record"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
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
	return &ReconcileCluster{
		Client:        mgr.GetClient(),
		scheme:        mgr.GetScheme(),
		EventRecorder: mgr.GetRecorder("ClusterController"),
	}
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

				machineList, err := getClusterMachineList(mgr.GetClient(), a.Meta.GetName())
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
	record.EventRecorder
}

// Reconcile reads that state of the cluster for a Cluster object and makes changes based on the state read
// and what is in the Cluster.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=cluster.sds.samsung.com,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=machine.sds.samsung.com,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("cluster Controller Reconcile()")

	// Fetch the Cluster instance
	clusterInstance := &clusterv1alpha1.Cluster{}
	err := r.Get(context.TODO(), request.NamespacedName, clusterInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if clusterInstance.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, add the finalizer and update the object.
		if !util.ContainsString(clusterInstance.ObjectMeta.Finalizers, clusterv1alpha1.ClusterFinalizer) {
			clusterInstance.ObjectMeta.Finalizers =
				append(clusterInstance.ObjectMeta.Finalizers, clusterv1alpha1.ClusterFinalizer)

			if err := r.Update(context.Background(), clusterInstance); err != nil {
				return reconcile.Result{Requeue: true}, nil
			}
		}
	} else {
		// The object is being deleted
		if util.ContainsString(clusterInstance.ObjectMeta.Finalizers, clusterv1alpha1.ClusterFinalizer) {
			// update status to "deleting"
			clusterInstance.Status.Phase = common.DeletingResourcePhase
			err = r.updateStatus(clusterInstance, corev1.EventTypeNormal,
				common.ResourceStateChange, common.MessageResourceStateChange,
				clusterInstance.GetName(), common.DeletingResourcePhase)
			if err != nil {
				log.Error(err, "could not update cluster status")
				return reconcile.Result{}, err
			}

			// there is a finalizer so we check if there are any machines left
			machineList, err := getClusterMachineList(r.Client, clusterInstance.GetName())
			if err != nil {
				log.Error(err, "could not list Machines")
				return reconcile.Result{}, err
			}

			// delete machines unless they are already being deleted and requeue
			if len(machineList.Items) > 0 {
				for _, machineInstance := range machineList.Items {
					if machineInstance.Status.Phase != common.DeletingResourcePhase {
						err = r.Client.Delete(context.Background(), &machineInstance)
						if err != nil {
							log.Error(err, "could not delete Machine "+machineInstance.GetName())
						}
						return reconcile.Result{Requeue: true}, err
					}
				}
			}

			// if no Machines left to be deleted
			// remove our finalizer from the list and update it.
			clusterInstance.ObjectMeta.Finalizers =
				util.RemoveString(clusterInstance.ObjectMeta.Finalizers, clusterv1alpha1.ClusterFinalizer)
			if err := r.Update(context.Background(), clusterInstance); err != nil {
				return reconcile.Result{Requeue: true}, nil
			}
		}
	}

	machineList, err := getClusterMachineList(r.Client, clusterInstance.GetName())
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
		log.Error(err, "could not get status of all cluster machines")
		clusterInstance.Status.ErrorMessage = "Cluster machines are in inconsistent phase" +
			" - all must be either Ready or the same one phase"
		clusterInstance.Status.ErrorReason = common.UnsupportedChangeClusterError
		clusterInstance.Status.Phase = common.ErrorResourcePhase

		err = r.updateStatus(clusterInstance, corev1.EventTypeWarning,
			common.ErrResourceFailed, common.MessageResourceFailed, clusterInstance.GetName())
		if err != nil {
			log.Error(err, "could not update cluster status")
		}

		return reconcile.Result{}, err
	}

	clusterInstance.Status.Phase = clusterStatus
	err = r.updateStatus(clusterInstance, corev1.EventTypeNormal,
		common.ResourceStateChange, common.MessageResourceStateChange, clusterInstance.GetName(), clusterStatus)
	if err != nil {
		log.Error(err, "could not update cluster status")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileCluster) updateStatus(clusterInstance *clusterv1alpha1.Cluster, eventType string,
	event common.ControllerEvents, eventMessage common.ControllerEvents, args ...interface{}) error {
	clusterInstance.Status.LastUpdated = &metav1.Time{Time: time.Now()}

	r.Eventf(clusterInstance, eventType,
		string(event), string(eventMessage), args)

	return r.Status().Update(context.Background(), clusterInstance)
}

func getClusterMachineList(c client.Client, clusterName string) (*clusterv1alpha1.MachineList, error) {
	machineList := &clusterv1alpha1.MachineList{}
	err := c.List(
		context.TODO(),
		&client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(
				"spec.clustername",
				clusterName),
		},
		machineList)
	if err != nil {
		return nil, err
	}

	return machineList, nil
}
