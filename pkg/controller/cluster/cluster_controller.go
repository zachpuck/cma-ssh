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
	"time"

	"github.com/golang/glog"
	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/common"
	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
	"github.com/samsung-cnct/cma-ssh/pkg/cert"
	"github.com/samsung-cnct/cma-ssh/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
	err = c.Watch(&source.Kind{Type: &clusterv1alpha1.CnctCluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &clusterv1alpha1.CnctMachine{}},
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
// +kubebuilder:rbac:groups=cluster.cnct.sds.samsung.com,resources=cnctclusters;cnctmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the Cluster instance
	cluster := &clusterv1alpha1.CnctCluster{}

	err := r.Get(context.Background(), request.NamespacedName, cluster)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return. Created objects are automatically garbage collected.
			//log.Error(err, "could not find cluster", "cluster", request)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		glog.Errorf("error reading object cluster %s: %q", request.Name, err)
		return reconcile.Result{}, err
	}

	if !cluster.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is being deleted
		if util.ContainsString(cluster.ObjectMeta.Finalizers, clusterv1alpha1.ClusterFinalizer) {
			glog.Info("deleting cluster...")

			// update status to "deleting"
			if cluster.Status.Phase != common.StoppingClusterPhase {
				cluster.Status.Phase = common.StoppingClusterPhase
				err = r.updateStatus(cluster, corev1.EventTypeNormal,
					common.ResourceStateChange, common.MessageResourceStateChange,
					cluster.GetName(), common.StoppingClusterPhase)
				if err != nil {
					glog.Errorf("could not update status of cluster %q: %q", cluster.GetName(), err)
					return reconcile.Result{}, err
				}
			}

			// there is a finalizer so we check if there are any machines left
			machineList, err := util.GetClusterMachineList(r.Client, cluster.GetName())
			if err != nil {
				glog.Errorf("could not list Machines for object cluster %q: %q", cluster.GetName(), err)
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
							glog.Errorf("could not delete machine %q for cluster %q: %q",
								machine.GetName(), cluster.GetName(), err)
						}
					}
				}

				return reconcile.Result{}, err
			}

			// if no Machines left to be deleted
			// set phase to deleted so secrets can be deleted and finalizer
			// can be removed
			cluster.Status.Phase = common.StoppingClusterPhase
		}
	}

	switch cluster.Status.Phase {
	case "":
		if err := createClusterSecrets(r.Client, cluster); err != nil {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cluster-private-key",
					Namespace: cluster.Namespace,
				},
			}

			if secret == nil {
				glog.Errorf("Failed to create cluster secrets: %s\n", err)
				return reconcile.Result{Requeue: true}, err
			}
		}
		glog.Info("cluster secrets created")
		cluster.Status.Phase = common.RunningClusterPhase
		cluster.ObjectMeta.Finalizers = append(cluster.ObjectMeta.Finalizers, clusterv1alpha1.ClusterFinalizer)
		err = r.updateStatus(
			cluster,
			corev1.EventTypeNormal,
			common.ResourceStateChange,
			common.MessageResourceStateChange,
			cluster.GetName(),
			common.RunningClusterPhase,
		)
		if err != nil {
			glog.Errorf("could not update cluster %q status: %q", cluster.GetName(), err)
			return reconcile.Result{Requeue: true}, err
		}
	case common.StoppingClusterPhase:
		if err := deleteClusterSecrets(r.Client, cluster); err != nil {
			glog.Errorf("Failed to delete cluster secrets: %s\n", err)
			return reconcile.Result{Requeue: true}, err
		}
		glog.Info("cluster secrets deleted")
		cluster.ObjectMeta.Finalizers =
			util.RemoveString(cluster.ObjectMeta.Finalizers, clusterv1alpha1.ClusterFinalizer)

		glog.Info("cluster is deleted")
		cluster.Status.Phase = ""
		return reconcile.Result{}, r.Update(context.Background(), cluster)
	}

	machineList, err := util.GetClusterMachineList(r.Client, cluster.GetName())
	if err != nil {
		glog.Errorf("could not list Machines for cluster %q: %q", cluster.GetName(), err)
		return reconcile.Result{}, err
	}

	clusterStatus, apiEndpoint := util.GetStatus(machineList)
	if cluster.Status.Phase != clusterStatus || cluster.Status.APIEndpoint != apiEndpoint {
		cluster.Status.Phase = clusterStatus
		cluster.Status.APIEndpoint = apiEndpoint
		err = r.updateStatus(cluster, corev1.EventTypeNormal,
			common.ResourceStateChange, common.MessageResourceStateChange, cluster.GetName(), clusterStatus)
		if err != nil {
			glog.Errorf("could not update cluster %q status: %q", cluster.GetName(), err)
		}
	}
	return reconcile.Result{}, err
}

func createClusterSecrets(k8sClient client.Client, cluster *clusterv1alpha1.CnctCluster) error {
	bundle, err := cert.NewCABundle()

	if err != nil {
		glog.Error(err)
		return err
	}

	dataMap := map[string][]byte{}
	bundle.MergeWithMap(dataMap)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-private-key",
			Namespace: cluster.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: dataMap,
	}

	return k8sClient.Create(context.Background(), secret)
}

// how does one GET a secret from k8s?
func deleteClusterSecrets(k8sClient client.Client, cluster *clusterv1alpha1.CnctCluster) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-private-key",
			Namespace: cluster.Namespace,
		},
	}

	if secret != nil {
		return k8sClient.Delete(context.Background(), secret)
	}

	return nil
}

func (r *ReconcileCluster) updateStatus(clusterInstance *clusterv1alpha1.CnctCluster, eventType string,
	event common.ControllerEvents, eventMessage common.ControllerEvents, args ...interface{}) error {

	clusterFreshInstance := &clusterv1alpha1.CnctCluster{}
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
