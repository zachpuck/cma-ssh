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
	"fmt"
	"github.com/samsung-cnct/cma-ssh/pkg/util/k8sutil"
	"time"

	"github.com/masterminds/semver"
	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/common"
	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
	"github.com/samsung-cnct/cma-ssh/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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

type backgroundMachineOp func(r *ReconcileMachine, machineInstance *clusterv1alpha1.CnctMachine) (string, error)

// Add creates a new Machine Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileMachine{
		Client:        mgr.GetClient(),
		scheme:        mgr.GetScheme(),
		EventRecorder: mgr.GetRecorder("MachineController"),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("machine-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Machine
	err = c.Watch(&source.Kind{Type: &clusterv1alpha1.CnctMachine{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &clusterv1alpha1.CnctCluster{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: util.ClusterToMachineMapper{Client: mgr.GetClient()}})
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
	record.EventRecorder
}

// Reconcile reads that stamakte of the cluster for a Machine object and makes changes based on the state read
// and what is in the Machine.Spec
// +kubebuilder:rbac:groups=cnctmachine.cnct.sds.samsung.com,resources=cnctmachines;cnctclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileMachine) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("machine Controller Reconcile()")

	// Fetch the Machine machine
	machineInstance := &clusterv1alpha1.CnctMachine{}
	err := r.Get(context.Background(), request.NamespacedName, machineInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// log.Error(err, "could not find machine", "machine", request)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "could not read machine", "machine", request)
		return reconcile.Result{}, err
	}

	clusterInstance, err := getCluster(r.Client, machineInstance.GetNamespace(), machineInstance.Spec.ClusterRef)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			log.Error(err, "could not find cluster "+machineInstance.Spec.ClusterRef)

			machineInstance.Status.Phase = common.ErrorMachinePhase
			err = r.updateStatus(machineInstance, corev1.EventTypeWarning, common.ErrResourceFailed,
				common.MessageResourceFailed, machineInstance.GetName())
			if err != nil {
				log.Error(err, "could not update status of object machine", "machine", machineInstance)
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, err
		}
		// Error reading the object - requeue the request.
		log.Error(err, "error reading object machine", "machine", machineInstance)
		return reconcile.Result{}, err
	}

	// if we are not being deleted
	if machineInstance.DeletionTimestamp.IsZero() {
		if machineInstance.Status.Phase == common.ReadyMachinePhase {
			// Object is currently ready, so we need to check if
			// the state change came in from cluster version change

			// build semvers
			machineKuberneteVersion, err := semver.NewVersion(machineInstance.Status.KubernetesVersion)
			if err != nil {
				log.Error(err, "could not parse object machine kubernetes version", "machine", machineInstance)
				return reconcile.Result{}, err
			}

			clusterKuberneteVersion, err := semver.NewVersion(clusterInstance.Spec.KubernetesVersion)
			if err != nil {
				log.Error(err, "could not parse object cluster kubernetes version", "cluster", clusterInstance)
				return reconcile.Result{}, err
			}

			// if cluster object kubernetes version is not equal to machine kubernetes version,
			// trigger an upgrade
			log.Info("Checking cluster version to see if upgrade is needed for " + machineInstance.GetName())
			if !clusterKuberneteVersion.Equal(machineKuberneteVersion) {
				log.Info("will upgrade " + machineInstance.GetName() + " to " + clusterKuberneteVersion.String())
				return r.handleUpgrade(machineInstance, clusterInstance)
			} else {
				log.Info("No upgrade is needed for " + machineInstance.GetName())
			}

		} else if machineInstance.Status.Phase == common.ErrorMachinePhase {
			// if object is in error state, just ignore and move on
			return reconcile.Result{}, nil
		} else if machineInstance.Status.Phase == common.UpgradingMachinePhase {
			// if object is currently updating, ignore and move on
			return reconcile.Result{}, nil
		} else {
			// if not an error, in progress update, or an update request, we must be
			// creating a new machine. Trigger bootstrap
			return r.handleCreate(machineInstance, clusterInstance)
		}
	} else {
		// The object is being deleted, do a node delete
		return r.handleDelete(machineInstance)
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileMachine) handleDelete(machineInstance *clusterv1alpha1.CnctMachine) (reconcile.Result, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("machine Controller handleDelete()")

	// if already deleting, ignore
	if machineInstance.Status.Phase == common.DeletingMachinePhase {
		log.Info("Delete: Already deleting...")
		return reconcile.Result{}, nil
	}

	// get cluster status to determine whether we should proceed,
	// i.e. if there is a create in progress, we wait for it to either finish or error
	clusterInstance, err := getCluster(r.Client, machineInstance.GetNamespace(), machineInstance.Spec.ClusterRef)
	if err != nil {
		return reconcile.Result{}, err
	}
	machineList, err := util.GetClusterMachineList(r.Client, clusterInstance.GetName())
	if err != nil {
		log.Error(err, "could not list Machines")
		return reconcile.Result{}, err
	}
	if !util.IsReadyForDeletion(machineList) {
		log.Info("Delete: Waiting for cluster to finish reconciling")
		return reconcile.Result{Requeue: true}, nil
	}

	if util.ContainsString(machineInstance.Finalizers, clusterv1alpha1.MachineFinalizer) {
		// update status to "deleting"
		machineInstance.Status.Phase = common.DeletingMachinePhase
		err := r.updateStatus(machineInstance, corev1.EventTypeNormal,
			common.ResourceStateChange, common.MessageResourceStateChange,
			machineInstance.GetName(), common.DeletingMachinePhase)
		if err != nil {
			log.Error(err, "could not update status of machine", "machine", machineInstance)
			return reconcile.Result{}, err
		}

		// start delete process
		r.backgroundRunner(doDelete, machineInstance, "handleDelete")

	}

	return reconcile.Result{}, nil
}

func (r *ReconcileMachine) handleUpgrade(machineInstance *clusterv1alpha1.CnctMachine, clusterInstance *clusterv1alpha1.CnctCluster) (reconcile.Result, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("machine Controller handleUpgrade()")

	// if already upgrading, move on
	if machineInstance.Status.Phase == common.UpgradingMachinePhase {
		return reconcile.Result{}, nil
	}

	// get cluster status to determine whether we should proceed,
	// i.e. if there is a create in progress, we wait for it to either finish or error
	clusterInstance, err := getCluster(r.Client, machineInstance.GetNamespace(), machineInstance.Spec.ClusterRef)
	if err != nil {
		return reconcile.Result{}, err
	}
	machineList, err := util.GetClusterMachineList(r.Client, clusterInstance.GetName())
	if err != nil {
		log.Error(err, "could not list Machines")
		return reconcile.Result{}, err
	}
	// if not ok to upgrade with error, return and do not requeue
	ok, err := util.IsReadyForUpgrade(machineList)
	if err != nil {
		log.Error(err, "cannot upgrade")
		return reconcile.Result{}, nil
	}
	// if not ok to upgrade, try later
	if !ok {
		log.Info("Upgrade: Waiting for cluster to finish reconciling")
		return reconcile.Result{Requeue: true}, nil
	}

	// update status to "upgrading"
	machineInstance.Status.Phase = common.UpgradingMachinePhase
	err = r.updateStatus(machineInstance, corev1.EventTypeNormal,
		common.ResourceStateChange, common.MessageResourceStateChange,
		machineInstance.GetName(), common.UpgradingMachinePhase)
	if err != nil {
		log.Error(err, "could not update status of machine", "machine", machineInstance)
		return reconcile.Result{}, err
	}

	// otherwise start upgrade process
	r.backgroundRunner(doUpgrade, machineInstance, "handleUpgrade")

	return reconcile.Result{}, nil
}

func (r *ReconcileMachine) handleCreate(machineInstance *clusterv1alpha1.CnctMachine, clusterInstance *clusterv1alpha1.CnctCluster) (reconcile.Result, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("machine Controller handleCreate()")

	if machineInstance.Status.Phase == common.ProvisioningMachinePhase {
		return reconcile.Result{}, nil
	}

	// Add the finalizer
	if !util.ContainsString(machineInstance.Finalizers, clusterv1alpha1.MachineFinalizer) {
		machineInstance.Finalizers =
			append(machineInstance.Finalizers, clusterv1alpha1.MachineFinalizer)
	}

	// update status to "creating"
	machineInstance.Status.Phase = common.ProvisioningMachinePhase
	machineInstance.Status.KubernetesVersion = clusterInstance.Spec.KubernetesVersion
	err := r.updateStatus(machineInstance, corev1.EventTypeNormal,
		common.ResourceStateChange, common.MessageResourceStateChange,
		machineInstance.GetName(), common.ProvisioningMachinePhase)
	if err != nil {
		log.Error(err, "could not update status of machine", "machine", machineInstance)
		return reconcile.Result{}, err
	}

	// start bootstrap process
	r.backgroundRunner(doBootstrap, machineInstance, "handleCreate")

	return reconcile.Result{}, nil
}

func (r *ReconcileMachine) updateStatus(machineInstance *clusterv1alpha1.CnctMachine, eventType string,
	event common.ControllerEvents, eventMessage common.ControllerEvents, args ...interface{}) error {

	machineFreshInstance := &clusterv1alpha1.CnctMachine{}
	err := r.Get(
		context.Background(),
		client.ObjectKey{
			Namespace: machineInstance.GetNamespace(),
			Name:      machineInstance.GetName(),
		}, machineFreshInstance)
	if err != nil {
		return err
	}

	machineFreshInstance.Finalizers = machineInstance.Finalizers
	machineFreshInstance.Status.Phase = machineInstance.Status.Phase
	machineFreshInstance.Status.KubernetesVersion = machineInstance.Status.KubernetesVersion
	machineFreshInstance.Status.LastUpdated = &metav1.Time{Time: time.Now()}

	err = r.Update(context.Background(), machineFreshInstance)
	if err != nil {
		return err
	}

	r.Eventf(machineFreshInstance, eventType,
		string(event), string(eventMessage), args...)

	return nil
}

func getCluster(c client.Client, namespace string, clusterName string) (*clusterv1alpha1.CnctCluster, error) {

	clusterKey := client.ObjectKey{
		Namespace: namespace,
		Name:      clusterName,
	}

	clusterInstance := &clusterv1alpha1.CnctCluster{}
	err := c.Get(context.Background(), clusterKey, clusterInstance)
	if err != nil {
		return nil, err
	}

	return clusterInstance, nil
}

func doBootstrap(r *ReconcileMachine, machineInstance *clusterv1alpha1.CnctMachine) (string, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("machine Controller doBootstrap()")

	// Setup bootstrap repo
	_, cmd, err := RunSshCommand(r.Client, machineInstance, InstallBootstrapRepo, make(map[string]string))
	if err != nil {
		return cmd, err
	}

	// install local nginx proxy
	_, cmd, err = RunSshCommand(r.Client, machineInstance, InstallNginx, make(map[string]string))
	if err != nil {
		return cmd, err
	}

	// install docker
	_, cmd, err = RunSshCommand(r.Client, machineInstance, InstallDocker, make(map[string]string))
	if err != nil {
		return cmd, err
	}

	// install kubernetes components
	_, cmd, err = RunSshCommand(r.Client, machineInstance, InstallKubernetes, make(map[string]string))
	if err != nil {
		return cmd, err
	}

	// if this is a master, proceed with bootstrap
	if util.ContainsRole(machineInstance.Spec.Roles, common.MachineRoleMaster) {
		// run kubeadm init
		_, cmd, err = RunSshCommand(r.Client, machineInstance, KubeadmInit, make(map[string]string))
		if err != nil {
			return cmd, err
		}

		// get kubeconfig
		kubeConfig, cmd, err := RunSshCommand(r.Client, machineInstance, GetKubeConfig, make(map[string]string))
		if err != nil {
			return cmd, err
		}

		// create kubeconfig secret with cluster as controller reference
		clusterInstance, err := getCluster(r.Client, machineInstance.GetNamespace(), machineInstance.Spec.ClusterRef)
		if err != nil {
			return "getCluster()", err
		}

		err = k8sutil.CreateKubeconfigSecret(r.Client, clusterInstance, r.scheme, kubeConfig)
		if err != nil {
			return "k8sutil.CreateKubeconfigSecret()", err
		}

	} else if util.ContainsRole(machineInstance.Spec.Roles, common.MachineRoleWorker) {
		// on worker, see if master is able to run kubeadm
		// if it is, run kubeadm token create and use the token to
		// do kubeadm join.
		// otherwise wait for a bit and try again.

		// get machine list
		machineList := &clusterv1alpha1.CnctMachineList{}
		err := r.List(
			context.Background(),
			&client.ListOptions{LabelSelector: labels.Everything()},
			machineList)
		if err != nil {
			return "get machine list", err
		}

		masterMachine, err := util.GetMaster(machineList.Items)
		if err != nil {
			return "util.GetMaster()", err
		}

		var token []byte
		err = util.Retry(60, 10*time.Second, func() error {
			// run kubeadm create token on master machine, get token back
			log.Info("Trying to get kubeadm token from master...")
			token, cmd, err = RunSshCommand(r.Client, masterMachine, KubeadmTokenCreate, make(map[string]string))
			if err != nil {
				log.Info("Waiting for kubeadm to be able to create a token",
					"machine", masterMachine)
				return err
			}

			log.Info("Got master kubeadm token: " + string(token[:]))
			return nil
		})

		// run kubeadm join on worker machine
		_, cmd, err = RunSshCommand(r.Client, machineInstance,
			KubeadmJoin, map[string]string{"token": string(token[:]), "master": masterMachine.Spec.SshConfig.Host})
		if err != nil {
			return cmd, err
		}
	}

	// Set status to ready
	machineInstance.Status.Phase = common.ReadyMachinePhase
	err = r.updateStatus(machineInstance, corev1.EventTypeNormal,
		common.ResourceStateChange, common.MessageResourceStateChange,
		machineInstance.GetName(), common.ReadyMachinePhase)
	if err != nil {
		return "updateStatus()", err
	}
	return "doBootstrap()", err
}

func doUpgrade(r *ReconcileMachine, machineInstance *clusterv1alpha1.CnctMachine) (string, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("machine Controller doUpgrade()")

	// get the cluster instance
	clusterInstance, err := getCluster(r.Client, machineInstance.GetNamespace(), machineInstance.Spec.ClusterRef)
	if err != nil {
		return "getCluster()", err
	}

	// if master, do the upgrade.
	// otherwise wait for master to finish upgrade then
	// proceed with upgrade
	// master is done upgrading when it is in ready status and
	// its status kubernetes version matches clusters kubernetes version
	if util.ContainsRole(machineInstance.Spec.Roles, common.MachineRoleMaster) {
		log.Info("running upgrade on master " + machineInstance.GetName())
		_, cmd, err := RunSshCommand(r.Client, machineInstance,
			UpgradeMaster, make(map[string]string))
		if err != nil {
			return cmd, err
		}

	} else {
		log.Info("running upgrade on worker " + machineInstance.GetName())
		err = util.Retry(60, 10*time.Second, func() error {
			// get list of machines
			machineList, err := util.GetClusterMachineList(r.Client, clusterInstance.GetName())
			if err != nil {
				log.Error(err, "could not list Machines")
				return err
			}

			masterMachine, err := util.GetMaster(machineList)
			if err != nil {
				log.Error(err, "could not get master instance")
				return err
			}

			if masterMachine.Status.Phase != common.ReadyMachinePhase {
				log.Info("master " + masterMachine.GetName() + " not ready, will retry " + machineInstance.GetName())
				return fmt.Errorf("master is not ready")
			}

			if masterMachine.Status.KubernetesVersion != clusterInstance.Spec.KubernetesVersion {
				log.Info("master " + masterMachine.GetName() +
					" not done with upgrade, will retry " + machineInstance.GetName())
				return fmt.Errorf("master is not done with upgrade")
			}

			log.Info("master phase: " + string(masterMachine.Status.Phase) +
				" master version: " + masterMachine.Status.KubernetesVersion)
			return nil
		})

		machineList, err := util.GetClusterMachineList(r.Client, clusterInstance.GetName())
		if err != nil {
			log.Error(err, "could not list Machines")
			return "util.GetClusterMachineList", err
		}

		masterMachine, err := util.GetMaster(machineList)
		if err != nil {
			log.Error(err, "could not get master instance")
			return "util.GetMaster", err
		}

		// get admin kubeconfig
		kubeConfig, cmd, err := RunSshCommand(r.Client, masterMachine, GetKubeConfig, make(map[string]string))
		if err != nil {
			return cmd, err
		}

		// run node upgrade
		_, cmd, err = RunSshCommand(r.Client, machineInstance,
			UpgradeNode, map[string]string{"admin.conf": string(kubeConfig[:])})
		if err != nil {
			return cmd, err
		}
	}

	machineInstance.Status.Phase = common.ReadyMachinePhase
	machineInstance.Status.KubernetesVersion = clusterInstance.Spec.KubernetesVersion

	err = r.updateStatus(machineInstance, corev1.EventTypeNormal,
		common.ResourceStateChange, common.MessageResourceStateChange,
		machineInstance.GetName(), common.ReadyMachinePhase)
	if err != nil {
		return "updateStatus()", err
	}

	return "doUpgrade()", nil
}

func doDelete(r *ReconcileMachine, machineInstance *clusterv1alpha1.CnctMachine) (string, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("machine Controller doDelete()")

	log.Info("Starting Delete for machine " + machineInstance.GetName())

	// there is a few possibilities here:
	// 1. Cluster is being deleted. We can just delete this machine
	// 2. Worker machine is being deleted. We should drain the machine first, and then delete it
	// 3. Master machine is being deleted. We should issue a warning, and then delete it.
	// get the cluster instance
	clusterInstance, err := getCluster(r.Client, machineInstance.GetNamespace(), machineInstance.Spec.ClusterRef)
	if err != nil {
		return "getCluster()", err
	}

	// Cluster is NOT being deleted.
	if clusterInstance.Status.Phase != common.StoppingClusterPhase {
		// get the master machine
		machineList, err := util.GetClusterMachineList(r.Client, clusterInstance.GetName())
		if err != nil {
			log.Error(err, "could not list Machines")
			return "util.GetClusterMachineList", err
		}
		masterMachine, err := util.GetMaster(machineList)
		if err != nil {
			log.Error(err, "could not get master instance")
			return "util.GetMaster", err
		}

		// TODO: this will need to be handled better
		if masterMachine.GetName() == machineInstance.GetName() {
			log.Info("WARNING!!! DELETING MASTER, CLUSTER WILL NOT FUNCTION WITHOUT NEW MASTER AND FULL RESET")
		}

		// get admin kubeconfig
		kubeConfig, cmd, err := RunSshCommand(r.Client, masterMachine, GetKubeConfig, make(map[string]string))
		if err != nil {
			return cmd, err
		}

		// run node drain
		_, cmd, err = RunSshCommand(r.Client, machineInstance,
			DrainAndDeleteNode, map[string]string{"admin.conf": string(kubeConfig[:])})
		if err != nil {
			return cmd, err
		}
	}

	// run delete command
	_, cmd, err := RunSshCommand(r.Client, machineInstance, DeleteNode, make(map[string]string))
	if err != nil {
		log.Error(err, "failed to clean up physical node for machine "+machineInstance.GetName()+
			". Manual cleanup might be required for "+machineInstance.Spec.SshConfig.Host)
	}

	machineInstance.Finalizers =
		util.RemoveString(machineInstance.Finalizers, clusterv1alpha1.MachineFinalizer)
	if err := r.updateStatus(machineInstance, corev1.EventTypeNormal,
		common.ResourceStateChange, common.MessageResourceStateChange,
		machineInstance.GetName(), common.DeletingMachinePhase); err != nil {
		log.Error(err, "failed to update status for machine "+machineInstance.GetName())
	}

	return cmd, err
}

func (r *ReconcileMachine) backgroundRunner(op backgroundMachineOp,
	machineInstance *clusterv1alpha1.CnctMachine, operationName string) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("machine Controller backgroundRunner()")

	type commandError struct {
		Err error
		Cmd string
	}

	// start bootstrap command (or pre upgrade etc)
	opResult := make(chan commandError)
	timer := time.NewTimer(20 * time.Minute)

	go func(ch chan<- commandError) {
		cmd, err := op(r, machineInstance)
		ch <- commandError{Err: err, Cmd: cmd}
		close(opResult)
	}(opResult)

	go func() {
		timedOut := false
		select {
		// cluster operation timeouts shouldn't take longer than 10 minutes
		case <-timer.C:
			err := fmt.Errorf("operation %s timed out for machine %s",
				operationName, machineInstance.GetNamespace())
			log.Error(err, "Provisioning operation timed out for object machine",
				"machine", machineInstance)

			timer.Stop()
			timedOut = true
		case result := <-opResult:
			// if finished with error
			if result.Err != nil {
				log.Error(result.Err, "could not complete object machine pre-start step of "+operationName,
					"machine", machineInstance, "command", result.Cmd)

				// set error status
				machineInstance.Status.Phase = common.ErrorMachinePhase
				err := r.updateStatus(machineInstance, corev1.EventTypeWarning,
					common.ResourceStateChange, common.MessageResourceStateChange,
					machineInstance.GetName(), common.ErrorMachinePhase)
				if err != nil {
					log.Error(err, "could not update status of machine", "machine", machineInstance)
				}
			}
		}

		// if timed out, wait for operation to complete
		if timedOut {
			select {
			case result := <-opResult:
				log.Error(result.Err, "Timed out object machine pre-start step of "+operationName,
					"machine", machineInstance, "command", result.Cmd)
			}

			// set error status
			machineInstance.Status.Phase = common.ErrorMachinePhase
			err := r.updateStatus(machineInstance, corev1.EventTypeWarning,
				common.ResourceStateChange, common.MessageResourceStateChange,
				machineInstance.GetName(), common.ErrorMachinePhase)
			if err != nil {
				log.Error(err, "could not update status of machine", "machine", machineInstance)
			}
		}

	}()
}
