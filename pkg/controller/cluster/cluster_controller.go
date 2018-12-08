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
	"k8s.io/client-go/tools/record"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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

	err = c.Watch(&source.Kind{Type: &clusterv1alpha1.Machine{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: util.MachineToClusterMapper{Client: mgr.GetClient()}})
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
// +kubebuilder:rbac:groups=cluster.cnct.sds.samsung.com,resources=clusters;machines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("cluster Controller Reconcile()")

	// Fetch the Cluster instance
	clusterInstance := &clusterv1alpha1.Cluster{}
	err := r.Get(context.Background(), request.NamespacedName, clusterInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			//log.Error(err, "could not find cluster", "cluster", request)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "error reading object cluster", "cluster", request)
		return reconcile.Result{}, err
	}

	if clusterInstance.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, add the finalizer and update the object.
		if !util.ContainsString(clusterInstance.ObjectMeta.Finalizers, clusterv1alpha1.ClusterFinalizer) {
			clusterInstance.ObjectMeta.Finalizers =
				append(clusterInstance.ObjectMeta.Finalizers, clusterv1alpha1.ClusterFinalizer)
			clusterInstance.Status.Phase = common.ReconcilingClusterPhase

			err = r.updateStatus(clusterInstance, corev1.EventTypeNormal,
				common.ResourceStateChange, common.MessageResourceStateChange,
				clusterInstance.GetName(), common.ReconcilingClusterPhase)
			if err != nil {
				log.Error(err, "could not update status of cluster", "cluster", clusterInstance)
				return reconcile.Result{}, err
			}
		}

	} else {
		// The object is being deleted
		if util.ContainsString(clusterInstance.ObjectMeta.Finalizers, clusterv1alpha1.ClusterFinalizer) {
			// update status to "deleting"
			if clusterInstance.Status.Phase != common.StoppingClusterPhase {
				clusterInstance.Status.Phase = common.StoppingClusterPhase
				err = r.updateStatus(clusterInstance, corev1.EventTypeNormal,
					common.ResourceStateChange, common.MessageResourceStateChange,
					clusterInstance.GetName(), common.StoppingClusterPhase)
				if err != nil {
					log.Error(err, "could not update status of cluster", "cluster", clusterInstance)
					return reconcile.Result{}, err
				}
			}

			// there is a finalizer so we check if there are any machines left
			machineList, err := util.GetClusterMachineList(r.Client, clusterInstance.GetName())
			if err != nil {
				log.Error(err, "could not list Machines for object cluster", "cluster", clusterInstance)
				return reconcile.Result{}, err
			}

			// delete the machines
			if len(machineList) > 0 {
				for _, machine := range machineList {
					if machine.Status.Phase == common.DeletingMachinePhase {
						continue
					}

					err = r.Delete(context.Background(), &machine)
					if err != nil {
						if !errors.IsNotFound(err) {
							log.Error(err, "could not delete object machine for object cluster",
								"machine", machine, "cluster", clusterInstance)
						}
					}
				}

				return reconcile.Result{}, err
			}

			// if no Machines left to be deleted
			// remove our finalizer from the list and update it.
			clusterInstance.ObjectMeta.Finalizers =
				util.RemoveString(clusterInstance.ObjectMeta.Finalizers, clusterv1alpha1.ClusterFinalizer)
			return reconcile.Result{}, r.Update(context.Background(), clusterInstance)
		}
	}

	machineList, err := util.GetClusterMachineList(r.Client, clusterInstance.GetName())
	if err != nil {
		log.Error(err, "could not list Machines")
		return reconcile.Result{}, err
	}

	clusterStatus, apiEndpoint := util.GetStatus(machineList)
	if clusterInstance.Status.Phase != clusterStatus || clusterInstance.Status.APIEndpoint != apiEndpoint {
		clusterInstance.Status.Phase = clusterStatus
		clusterInstance.Status.APIEndpoint = apiEndpoint
		err = r.updateStatus(clusterInstance, corev1.EventTypeNormal,
			common.ResourceStateChange, common.MessageResourceStateChange, clusterInstance.GetName(), clusterStatus)
		if err != nil {
			log.Error(err, "could not update object cluster status", "cluster", clusterInstance)
		}
	}
	return reconcile.Result{}, err
}

func (r *ReconcileCluster) updateStatus(clusterInstance *clusterv1alpha1.Cluster, eventType string,
	event common.ControllerEvents, eventMessage common.ControllerEvents, args ...interface{}) error {

	clusterFreshInstance := &clusterv1alpha1.Cluster{}
	err := r.Get(
		context.Background(),
		client.ObjectKey{
			Namespace: clusterInstance.GetNamespace(),
			Name:      clusterInstance.GetName(),
		}, clusterFreshInstance)
	if err != nil {
		return err
	}

	clusterFreshInstance.Status.LastUpdated = &metav1.Time{Time: time.Now()}
	clusterFreshInstance.Status.Phase = clusterInstance.Status.Phase
	clusterFreshInstance.Status.APIEndpoint = clusterInstance.Status.APIEndpoint
	clusterFreshInstance.ObjectMeta.Finalizers = clusterInstance.ObjectMeta.Finalizers

	err = r.Update(context.Background(), clusterFreshInstance)
	if err != nil {
		return err
	}

	r.Eventf(clusterFreshInstance, eventType,
		string(event), string(eventMessage), args...)

	return nil
}
