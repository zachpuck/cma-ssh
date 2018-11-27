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

package machine

import (
	"context"
	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/common"
	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Machine Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
// USER ACTION REQUIRED: update cmd/manager/main.go to call this cluster.Add(mgr) to install this Controller
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileMachine{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("machine-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Machine
	err = c.Watch(&source.Kind{Type: &clusterv1alpha1.Machine{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create
	// Uncomment watch a Deployment created by Machine - change this for objects you create
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &clusterv1alpha1.Machine{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileMachine{}

// ReconcileMachine reconciles a Machine object
type ReconcileMachine struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Machine object and makes changes based on the state read
// and what is in the Machine.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.sds.samsung.com,resources=machines,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileMachine) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the Machine machine
	machine := &clusterv1alpha1.Machine{}
	err := r.Get(context.TODO(), request.NamespacedName, machine)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if machine.Status.Phase == "" {
		log.Printf("provisioning machine")
		// TODO: call ssh util to provision machine

		// checks if provision is complete
		r.machineInitChecker(machine)
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileMachine) updateMachinePhase(machine *clusterv1alpha1.Machine, status common.StatusPhase) error {
	machineCopy := machine.DeepCopy()

	machineCopy.Status.Phase = status

	err := r.Client.Update(context.TODO(), machineCopy)
	return err
}

func (r *ReconcileMachine) machineInitChecker(machine *clusterv1alpha1.Machine) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	done := make(chan bool)
	go func() {
		time.Sleep(300 * time.Second)
		done <- true
	}()

	for {
		select {
		case <-done:
			log.Printf("provisioning failed to complete in provided time")
			status := common.StatusPhase("Failed")
			r.updateMachinePhase(machine, status)
			return
		case <-ticker.C:
			found, kubeconfig := false, "foo" // TODO: ssh to machine and check if /etc/kubernetes/admin.conf exists
			if found {
				// TODO: create a secret for kubeconfig and save it to the cmc
				log.Printf(kubeconfig)
				status := common.StatusPhase("Provisioned")
				r.updateMachinePhase(machine, status)
			}
		}

	}
}
