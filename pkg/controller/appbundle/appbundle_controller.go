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

package appbundle

import (
	"context"
	"github.com/pkg/errors"
	addonsv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/addons/v1alpha1"
	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/common"
	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

var log = logf.Log.WithName("app bundle controller")

// Add creates a new AppBundle Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileAppBundle{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("appbundle-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to AppBundle
	err = c.Watch(&source.Kind{Type: &addonsv1alpha1.AppBundle{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileAppBundle{}

// ReconcileAppBundle reconciles a AppBundle object
type ReconcileAppBundle struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a AppBundle object and makes changes based on the state read
// and what is in the AppBundle.Spec
// +kubebuilder:rbac:groups=addons.cnct.sds.samsung.com,resources=appbundles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=addons.cnct.sds.samsung.com,resources=appbundles/status,verbs=get;update;patch
func (r *ReconcileAppBundle) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the AppBundle instance
	appBundle := &addonsv1alpha1.AppBundle{}
	err := r.Get(context.TODO(), request.NamespacedName, appBundle)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	log.Info("checking if app bundle is installed", "appBundle", appBundle.Name, "namespace", appBundle.Namespace)
	if appBundle.Status.Phase != addonsv1alpha1.InstalledAppBundlePhase {
		// Fetch the CnctCluster Instance
		var clusters clusterv1alpha1.CnctClusterList
		err = r.List(context.Background(), &client.ListOptions{Namespace: appBundle.Namespace}, &clusters)
		if err != nil {
			return reconcile.Result{}, err
		}
		if len(clusters.Items) == 0 {
			log.Info("cluster not found while attempting to install, retrying", "appBundle", appBundle.Name, "namespace", appBundle.Namespace)
			return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
		}
		cluster := clusters.Items[0]
		if cluster.Status.Phase == common.RunningClusterPhase {
			log.Info("installing", "appBundle", appBundle.Name, "cluster", cluster.Name)

			// create clientset for connecting to remote cluster
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

			// install app in remote cluster
			err = r.install(clientset, appBundle)
			if err != nil {
				return reconcile.Result{}, errors.Wrap(err, "failed to install appBundle in remote cluster")
			}

			// TODO: check if install job completed successfully, set phase to "Installing" until completed, or Errored

			// update status to installed
			appBundle.Status.Phase = addonsv1alpha1.InstalledAppBundlePhase
			err = r.Client.Update(context.Background(), appBundle)
			if err != nil {
				return reconcile.Result{}, errors.Wrap(err, "failed to update appBundle status")
			}
			return reconcile.Result{}, nil
		}
		log.Info("waiting for cluster running status to install app bundle", "cluster", cluster.Name)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}
	log.Info("app bundle already installed", "appbundle", appBundle.Name, "cluster", appBundle.Namespace)
	return reconcile.Result{}, nil
}

func (r *ReconcileAppBundle) install(clientset *kubernetes.Clientset, appBundle *addonsv1alpha1.AppBundle) error {
	// create namespace
	err := createNamespace(clientset)
	if err != nil {
		return errors.Wrap(err, "failed to create namespace for cma")
	}
	// create service account
	err = createServiceAccount(clientset)
	if err != nil {
		return errors.Wrap(err, "failed to create service account for cma")
	}
	// create cluster role binding
	err = createClusterRoleBinding(clientset)
	if err != nil {
		return errors.Wrap(err, "failed to create cluster role binding for cma")
	}
	// create install job
	err = createInstallJob(clientset, appBundle)
	if err != nil {
		return errors.Wrap(err, "failed to create appBundle Job")
	}
	return nil
}
