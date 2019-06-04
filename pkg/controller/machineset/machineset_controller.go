/*
Copyright 2019 Samsung SDS.

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

package machineset

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/common"
	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
	"github.com/samsung-cnct/cma-ssh/pkg/util"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	log = logf.Log.WithName("CnctMachineSet-controller")

	// controllerKind contains the schema.GroupVersionKind for this controller type.
	controllerKind = clusterv1alpha1.SchemeGroupVersion.WithKind("CnctMachineSet")

	// stateConfirmationTimeout is the amount of time allowed to wait for desired state.
	stateConfirmationTimeout = 10 * time.Second

	// stateConfirmationInterval is the amount of time between polling for the desired state.
	// The polling is against a local memory cache.
	stateConfirmationInterval = 100 * time.Millisecond
)

// Add creates a new MachineSet Controller and adds it to the Manager with default RBAC.
// The Manager will set fields on the Controller and start it when the Manager is started.
func Add(mgr manager.Manager) error {
	r := newReconciler(mgr)
	return add(mgr, r, r.MachineToMachineSets)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) *ReconcileMachineSet {
	return &ReconcileMachineSet{
		Client:        mgr.GetClient(),
		scheme:        mgr.GetScheme(),
		EventRecorder: mgr.GetRecorder("MachineSetController")}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler, mapFn handler.ToRequestsFunc) error {
	// Create a new controller
	c, err := controller.New("machineset-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to MachineSet
	err = c.Watch(&source.Kind{Type: &clusterv1alpha1.CnctMachineSet{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to Machines and reconcile the owner MachineSet
	err = c.Watch(&source.Kind{Type: &clusterv1alpha1.CnctMachine{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &clusterv1alpha1.CnctMachineSet{},
		})
	if err != nil {
		return err
	}

	// Watch for changes to Machines using a mapping function to MachineSets.
	// This watcher is required for use cases like adoption. In case a Machine doesn't have
	// a controller reference, it'll look for potential matching MachineSet to reconcile.
	return c.Watch(
		&source.Kind{Type: &clusterv1alpha1.CnctMachine{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn},
	)
	return nil
}

var _ reconcile.Reconciler = &ReconcileMachineSet{}

// ReconcileMachineSet reconciles a CnctMachineSet object
type ReconcileMachineSet struct {
	client.Client
	scheme *runtime.Scheme
	record.EventRecorder
}

// Reconcile reads the state of the cluster for a CnctMachineSet object and makes changes
// based on the state read and what is in the CnctMachineSet.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.cnct.sds.samsung.com,resources=cnctmachinesets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.cnct.sds.samsung.com,resources=cnctmachinesets/status,verbs=get;update;patch
func (r *ReconcileMachineSet) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Info("reconciling machineset", "request", request)
	// Fetch the CnctMachineSet instance
	instance := &clusterv1alpha1.CnctMachineSet{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// reconcile MachineSet logic
	r.reconcile(instance)

	return reconcile.Result{}, nil
}

// getCluster returns the Cluster associated with the MachineSet, if any.
func (r *ReconcileMachineSet) getCluster(ms *clusterv1alpha1.CnctMachineSet) (*clusterv1alpha1.CnctCluster, error) {
	cluster := &clusterv1alpha1.CnctCluster{}
	key := client.ObjectKey{
		Namespace: ms.Namespace,
		// assuming cluster name is the same as namespace
		Name: ms.Namespace,
	}
	if err := r.Client.Get(context.Background(), key, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

// reconcile the MachineSet
func (r *ReconcileMachineSet) reconcile(machineSet *clusterv1alpha1.CnctMachineSet) (reconcile.Result, error) {
	allMachines := &clusterv1alpha1.CnctMachineList{}
	if err := r.Client.List(context.Background(), client.InNamespace(machineSet.Namespace), allMachines); err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed to list machines")
	}

	// Validate MachineSet
	validMachineSet, err := validateMachineSet(machineSet)
	if !validMachineSet {
		log.Error(err, "MachineSet failed validation")
		return reconcile.Result{}, err
	}

	cluster, err := r.getCluster(machineSet)
	if err != nil {
		// Cluster might not be defined yet
		return reconcile.Result{}, err
	}

	// Set the ownerRef with foreground deletion if there is a linked cluster
	if cluster != nil && len(machineSet.OwnerReferences) == 0 {
		blockOwnerDeletion := true
		machineSet.OwnerReferences = append(machineSet.OwnerReferences, metav1.OwnerReference{
			APIVersion:         cluster.APIVersion,
			Kind:               cluster.Kind,
			Name:               cluster.Name,
			UID:                cluster.UID,
			BlockOwnerDeletion: &blockOwnerDeletion,
		})
	}

	// Add foregroundDeletion finalizer if MachineSet is not deleted
	if cluster != nil &&
		machineSet.ObjectMeta.DeletionTimestamp.IsZero() &&
		!util.ContainsString(machineSet.Finalizers, metav1.FinalizerDeleteDependents) {

		machineSet.Finalizers = append(machineSet.ObjectMeta.Finalizers, metav1.FinalizerDeleteDependents)

		if err := r.Client.Update(context.Background(), machineSet); err != nil {
			log.Error(err, "Failed to add finalizers to MachineSet", "MachineSet", machineSet.Name)
			return reconcile.Result{}, err
		}

		// Since adding the finalizer updates the object return to avoid later update issues
		return reconcile.Result{Requeue: true}, nil
	}

	// Return early if the MachineSet is deleted.
	if !machineSet.ObjectMeta.DeletionTimestamp.IsZero() {
		return reconcile.Result{}, nil
	}

	// Filter out irrelevant machines (filtering out/mismatch labels) and claim orphaned machines.
	filteredMachines := r.filterMachines(machineSet, allMachines)

	// Sync replicas
	syncErr := r.syncReplicas(machineSet, filteredMachines)
	if syncErr == nil {
		machineSet.Status.Phase = common.ReadyMachineSetPhase
	} else {
		log.Error(syncErr, "syncReplicas Error")
		machineSet.Status.Phase = common.ErrorMachineSetPhase
	}

	// Always Update status (regardless of syncErr)
	newStatus := calculateStatus(machineSet, filteredMachines)
	machineSet.Status = newStatus
	updateErr := r.updateStatus(machineSet, corev1.EventTypeNormal,
		common.ResourceStateChange, common.MessageResourceStateChange,
		machineSet.GetName(), common.ReadyMachineSetPhase)

	if syncErr != nil {
		return reconcile.Result{}, errors.Wrapf(syncErr, "failed to sync Machineset replicas")
	}
	if updateErr != nil {
		log.Error(updateErr, "could not update status of machineSet", "machineSet", *machineSet)
		return reconcile.Result{}, updateErr
	}
	return reconcile.Result{}, nil
}

// syncReplicas scales Machine resources up or down.
func (r *ReconcileMachineSet) syncReplicas(ms *clusterv1alpha1.CnctMachineSet, machines []*clusterv1alpha1.CnctMachine) error {
	diff := len(machines) - int(ms.Spec.Replicas)

	if diff < 0 {
		log.Info("Creating replicas!!!")
		diff *= -1
		log.Info("Too few replicas for", "machineset", *ms, "creating", diff)

		var machineList []*clusterv1alpha1.CnctMachine
		var errstrings []string
		for i := 0; i < diff; i++ {
			log.Info("Creating machine", "machine", i+1)
			machine := r.createMachine(ms)
			if err := r.Client.Create(context.Background(), machine); err != nil {
				log.Error(err, "Unable to create Machine", "Machine", machine.Name)
				errstrings = append(errstrings, err.Error())
				continue
			}

			machineList = append(machineList, machine)
		}

		if len(errstrings) > 0 {
			return errors.New(strings.Join(errstrings, "; "))
		}
		return r.waitForMachineCreation(machineList)
	} else if diff > 0 {
		log.Info("Deleting replicas!!!")
		log.Info("Too many replicas for", "machineset", *ms, "deleting", diff)
		deletePriorityFunc := randomDeletePolicy
		// Choose which Machines to delete.
		machinesToDelete := getMachinesToDeletePrioritized(machines, diff, deletePriorityFunc)

		// TODO: Add cap to limit concurrent delete calls.
		errCh := make(chan error, diff)
		var wg sync.WaitGroup
		wg.Add(diff)
		for _, machine := range machinesToDelete {
			go func(targetMachine *clusterv1alpha1.CnctMachine) {
				defer wg.Done()
				err := r.Client.Delete(context.Background(), targetMachine)
				if err != nil {
					log.Error(err, "Unable to delete Machine", "Machine", targetMachine.Name)
					errCh <- err
				}
			}(machine)
		}
		wg.Wait()

		select {
		case err := <-errCh:
			// all errors have been reported before and they're likely to be the same, so we'll only return the first one we hit.
			if err != nil {
				return err
			}
		default:
		}
		return r.waitForMachineDeletion(machinesToDelete)
	}

	return nil
}

// filterMachines filters out irrelevant machines and claims orphaned machines
func (r *ReconcileMachineSet) filterMachines(machineSet *clusterv1alpha1.CnctMachineSet, allMachines *clusterv1alpha1.CnctMachineList) []*clusterv1alpha1.CnctMachine {
	filteredMachines := make([]*clusterv1alpha1.CnctMachine, 0, len(allMachines.Items))
	for idx := range allMachines.Items {
		machine := &allMachines.Items[idx]
		if shouldExcludeMachine(machineSet, machine) {
			continue
		}
		// Attempt to adopt machine if it meets previous conditions and it has no controller references.
		if metav1.GetControllerOf(machine) == nil {
			if err := r.adoptOrphan(machineSet, machine); err != nil {
				log.Error(err, "Failed to adopt Machine into MachineSet", "machine", machine.Name,
					"machineSet", machineSet.Name)
				continue
			}
		}
		filteredMachines = append(filteredMachines, machine)
	}
	return filteredMachines
}

// adoptOrphan sets the MachineSet as a controller OwnerReference to the Machine.
func (r *ReconcileMachineSet) adoptOrphan(machineSet *clusterv1alpha1.CnctMachineSet, machine *clusterv1alpha1.CnctMachine) error {
	newRef := *metav1.NewControllerRef(machineSet, controllerKind)
	machine.OwnerReferences = append(machine.OwnerReferences, newRef)
	return r.Client.Update(context.Background(), machine)
}

// createMachine creates a Machine resource. The name of the newly created resource is going
// to be created by the API server, we set the generateName field.
func (r *ReconcileMachineSet) createMachine(machineSet *clusterv1alpha1.CnctMachineSet) *clusterv1alpha1.CnctMachine {
	gv := clusterv1alpha1.SchemeGroupVersion
	machine := &clusterv1alpha1.CnctMachine{
		TypeMeta: metav1.TypeMeta{
			Kind:       gv.WithKind("CnctMachine").Kind,
			APIVersion: gv.String(),
		},
		ObjectMeta: machineSet.Spec.MachineTemplate.ObjectMeta,
		Spec:       machineSet.Spec.MachineTemplate.Spec,
	}
	machine.ObjectMeta.GenerateName = fmt.Sprintf("%s-", machineSet.Name)
	machine.ObjectMeta.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(machineSet, controllerKind)}
	machine.Namespace = machineSet.Namespace
	return machine
}

func (r *ReconcileMachineSet) waitForMachineCreation(machineList []*clusterv1alpha1.CnctMachine) error {
	for _, machine := range machineList {
		pollErr := PollImmediate(stateConfirmationInterval, stateConfirmationTimeout, func() (bool, error) {
			key := client.ObjectKey{Namespace: machine.Namespace, Name: machine.Name}

			if err := r.Client.Get(context.Background(), key, &clusterv1alpha1.CnctMachine{}); err != nil {
				if apierrors.IsNotFound(err) {
					return false, nil
				}
				log.Error(err, "Failed to Get Machine while waiting for Machine creation")
				return false, err
			}
			return true, nil
		})

		if pollErr != nil {
			log.Error(pollErr, "Polling error while waiting for Machine creation")
			return errors.Wrap(pollErr, "failed waiting for machine object to be created")
		}
	}
	return nil
}

func (r *ReconcileMachineSet) waitForMachineDeletion(machineList []*clusterv1alpha1.CnctMachine) error {
	for _, machine := range machineList {
		pollErr := PollImmediate(stateConfirmationInterval, stateConfirmationTimeout, func() (bool, error) {
			m := &clusterv1alpha1.CnctMachine{}
			key := client.ObjectKey{Namespace: machine.Namespace, Name: machine.Name}

			err := r.Client.Get(context.Background(), key, m)
			if apierrors.IsNotFound(err) || !m.DeletionTimestamp.IsZero() {
				return true, nil
			}
			return false, err
		})

		if pollErr != nil {
			log.Error(pollErr, "Polling error while waiting for Machine deletion")
			return errors.Wrap(pollErr, "failed waiting for machine object to be deleted")
		}
	}
	return nil
}

// MachineToMachineSets is a handler.ToRequestsFunc used to enqeue requests for reconciliation
// for MachineSets that might adopt an orphaned Machine.
func (r *ReconcileMachineSet) MachineToMachineSets(o handler.MapObject) []reconcile.Request {
	result := []reconcile.Request{}

	m := &clusterv1alpha1.CnctMachine{}
	key := client.ObjectKey{Namespace: o.Meta.GetNamespace(), Name: o.Meta.GetName()}
	if err := r.Client.Get(context.Background(), key, m); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "Unable to retrieve Machine for possible MachineSet adoption",
				"machine", key)
		}
		return nil
	}

	// Check if the controller reference is already set and
	// return an empty result when one is found.
	for _, ref := range m.ObjectMeta.OwnerReferences {
		if ref.Controller != nil && *ref.Controller {
			return result
		}
	}
	mss := r.getMachineSetsForMachine(m)
	if len(mss) == 0 {
		log.Info("Found no MachineSet for Machine", "machine", m.Name)
		return nil
	}
	for _, ms := range mss {
		name := client.ObjectKey{Namespace: ms.Namespace, Name: ms.Name}
		result = append(result, reconcile.Request{NamespacedName: name})
	}
	return result
}
