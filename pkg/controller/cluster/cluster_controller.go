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

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/common"
	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
	"github.com/samsung-cnct/cma-ssh/pkg/cert"
)

var log = logf.Log.WithName("CnctCluster-controller")

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
	err = c.Watch(
		&source.Kind{Type: &clusterv1alpha1.CnctMachine{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
				c := mgr.GetClient()
				ns := a.Meta.GetNamespace()
				var clusters clusterv1alpha1.CnctClusterList
				err := c.List(context.Background(), &client.ListOptions{Namespace: ns}, &clusters)
				if err != nil {
					return nil
				}
				if len(clusters.Items) == 0 {
					return nil
				}
				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Name:      clusters.Items[0].Name,
							Namespace: clusters.Items[0].Namespace,
						},
					},
				}
			}),
		},
	)
	return err
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
		if apierrors.IsNotFound(err) {
			var machines clusterv1alpha1.CnctMachineList
			err := r.Client.List(context.Background(), &client.ListOptions{Namespace: request.Namespace}, &machines)
			if err != nil {
				return reconcile.Result{}, errors.Wrap(err, "could not list machines")
			}
			if len(machines.Items) != 0 {
				log.Info("cluster deletion pending on machine deletion", "pending machines", len(machines.Items))
				return reconcile.Result{RequeueAfter: 1 * time.Second}, nil
			}
			var secret corev1.Secret
			err = r.Get(context.Background(), client.ObjectKey{Name: "cluster-private-key", Namespace: request.Namespace}, &secret)
			if apierrors.IsNotFound(err) {
				// already deleted
			} else if err != nil {
				return reconcile.Result{}, errors.Wrap(err, "could not get secret")
			} else {
				errDelete := r.Delete(context.Background(), &secret)
				if errDelete != nil {
					return reconcile.Result{}, errors.Wrap(errDelete, "could not delete cluster secret")
				}
			}
			// Object not found, return. Created objects are automatically garbage collected.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "error reading object cluster", "request", request)
		return reconcile.Result{}, err
	}
	var machineList clusterv1alpha1.CnctMachineList
	err = r.List(context.Background(), &client.ListOptions{Namespace: request.Namespace}, &machineList)
	if err != nil {
		return reconcile.Result{}, err
	}
	machines := machineList.Items
	err = r.claimMachines(cluster, machines)
	if err != nil {
		return reconcile.Result{}, err
	}

	switch cluster.Status.Phase {
	case "":
		if err := createClusterSecrets(r.Client, cluster); err != nil {
			return reconcile.Result{Requeue: true}, err
		}
		log.Info("cluster secrets created")
		cluster.Status.Phase = common.ReconcilingClusterPhase
		err = r.updateStatus(
			cluster,
			corev1.EventTypeNormal,
			common.ResourceStateChange,
			common.MessageResourceStateChange,
			cluster.GetName(),
			common.ReconcilingClusterPhase,
		)
		if err != nil {
			log.Error(err, "could not update cluster status", "cluster", cluster)
			return reconcile.Result{Requeue: true}, err
		}
	case common.ReconcilingClusterPhase:
		var secret corev1.Secret
		err := r.Get(context.Background(), client.ObjectKey{Name: "cluster-private-key", Namespace: cluster.Namespace}, &secret)
		configData, ok := secret.Data[corev1.ServiceAccountKubeconfigKey]
		if !ok || len(configData) == 0 {
			return reconcile.Result{}, errors.New("no kubeconfig in secret")
		}
		config, err := clientcmd.NewClientConfigFromBytes(configData)
		if err != nil {
			return reconcile.Result{}, errors.Wrap(err, "could not create new client config from secret")
		}
		restConfig, err := config.ClientConfig()
		if err != nil {
			return reconcile.Result{}, errors.Wrap(err, "could not create rest client config")
		}
		clientset, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return reconcile.Result{}, errors.Wrap(err, "could not create a clientset")
		}
		// Copied from kubectl cluster-info
		serviceList, err := clientset.CoreV1().
			Services(metav1.NamespaceSystem).
			List(
				metav1.ListOptions{
					LabelSelector: "kubernetes.io/cluster-service=true",
				},
			)
		if apierrors.IsNotFound(err) {
			return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
		} else if err != nil {
			return reconcile.Result{}, errors.Wrap(err, "could not get service list")
		}
		if len(serviceList.Items) > 0 {
			cluster.Status.Phase = common.RunningClusterPhase
			if err := r.Update(context.Background(), cluster); err != nil {
				return reconcile.Result{}, errors.Wrap(err, "could not update cluster status")
			}
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, errors.New("cluster services are not running")
	}

	return reconcile.Result{}, err
}

func createClusterSecrets(k8sClient client.Client, cluster *clusterv1alpha1.CnctCluster) error {
	bundle, err := cert.NewCABundle()
	if err != nil {
		log.Error(err, "could not create a new ca cert bundle")
		return err
	}

	dataMap := map[string][]byte{}
	bundle.MergeWithMap(dataMap)
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-private-key",
			Namespace: cluster.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: dataMap,
	}
	return k8sClient.Create(context.Background(), &secret)
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

func (r *ReconcileCluster) claimMachines(cluster *clusterv1alpha1.CnctCluster, machines []clusterv1alpha1.CnctMachine) error {
	for i := range machines {
		if r.canClaim(cluster, &machines[i]) {
			machines[i].OwnerReferences = append(machines[i].OwnerReferences, *metav1.NewControllerRef(cluster, clusterv1alpha1.SchemeGroupVersion.WithKind("CnctCluster")))
			err := r.Update(context.Background(), &machines[i])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *ReconcileCluster) canClaim(cluster *clusterv1alpha1.CnctCluster, machine *clusterv1alpha1.CnctMachine) bool {
	for _, v := range machine.OwnerReferences {
		if v.Controller != nil && *v.Controller {
			return false
		}
	}
	return true
}
