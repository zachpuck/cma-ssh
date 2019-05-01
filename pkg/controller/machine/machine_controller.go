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
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	errs "github.com/pkg/errors"
	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/common"
	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
	"github.com/samsung-cnct/cma-ssh/pkg/cert"
	"github.com/samsung-cnct/cma-ssh/pkg/maas"
	"github.com/samsung-cnct/cma-ssh/pkg/ssh"
	"github.com/samsung-cnct/cma-ssh/pkg/util"
	"github.com/samsung-cnct/cma-ssh/pkg/util/k8sutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type backgroundMachineOp func(r *ReconcileMachine, machineInstance *clusterv1alpha1.CnctMachine, privateKey []byte) error

// Add creates a new Machine Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func AddWithActuator(mgr manager.Manager, maasClient maas.Client) error {
	return add(mgr, newReconciler(mgr, maasClient))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, maasClient maas.Client) reconcile.Reconciler {
	return &ReconcileMachine{
		Client:        mgr.GetClient(),
		scheme:        mgr.GetScheme(),
		EventRecorder: mgr.GetRecorder("MachineController"),
		MAASClient:    maasClient,
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
	MAASClient maas.Client
}

// Reconcile reads that state of the cluster for a Machine object and makes changes based on the state read
// and what is in the Machine.Spec
// +kubebuilder:rbac:groups=cluster.cnct.sds.samsung.com,resources=cnctmachines;cnctclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileMachine) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the Machine machine
	var machine clusterv1alpha1.CnctMachine
	if err := r.Get(context.Background(), request.NamespacedName, &machine); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// log.Error(err, "could not find machine", "machine", request)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		klog.Errorf("could not read machine %s: %q", request.Name, err)
		return reconcile.Result{}, err
	}

	var cluster clusterv1alpha1.CnctCluster
	if err := r.Get(context.Background(), types.NamespacedName{Name: machine.Spec.ClusterRef, Namespace: machine.Namespace}, &cluster); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			klog.Errorf("could not find cluster %s: %q", machine.Spec.ClusterRef, err)

			return reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Second}, nil
		}
		// Error reading the object - requeue the request.
		klog.Errorf("error reading object machine %s: %q", machine.GetName(), err)
		return reconcile.Result{}, err
	}

	if !machine.DeletionTimestamp.IsZero() {
		return r.handleDelete(&machine)
	}

	switch machine.Status.Phase {
	case common.ProvisioningMachinePhase:
		return r.handleWaitingForReady(&machine, &cluster)
	case common.DeletingMachinePhase:
		return r.handleDelete(&machine)
	case common.ErrorMachinePhase:
		return reconcile.Result{}, nil
	case "":
		return r.handleCreate(&machine, &cluster)
	default:
		return reconcile.Result{}, errs.Errorf("unknown phase %q", machine.Status.Phase)
	}
}

func (r *ReconcileMachine) handleDelete(machineInstance *clusterv1alpha1.CnctMachine) (reconcile.Result, error) {
	// get cluster status to determine whether we should proceed,
	// i.e. if there is a create in progress, we wait for it to either finish or error
	clusterInstance, err := getCluster(r.Client, machineInstance.GetNamespace(), machineInstance.Spec.ClusterRef)
	if err != nil {
		return reconcile.Result{}, err
	}
	machineList, err := util.GetClusterMachineList(r.Client, clusterInstance.GetName())
	if err != nil {
		klog.Errorf("could not list Machines: %q", err)
		return reconcile.Result{}, err
	}
	if !util.IsReadyForDeletion(machineList) {
		klog.Infof("Delete: Waiting for cluster %s to finish reconciling", machineInstance.Spec.ClusterRef)
		return reconcile.Result{Requeue: true}, nil
	}

	if util.ContainsString(machineInstance.Finalizers, clusterv1alpha1.MachineFinalizer) {
		// update status to "deleting"
		machineInstance.Status.Phase = common.DeletingMachinePhase
		err := r.updateStatus(machineInstance, corev1.EventTypeNormal,
			common.ResourceStateChange, common.MessageResourceStateChange,
			machineInstance.GetName(), common.DeletingMachinePhase)
		if err != nil {
			klog.Errorf("could not update status of machine %s: %q", machineInstance.GetName(), err)
			return reconcile.Result{}, err
		}

		if err := deleteMachine(r, machineInstance); err != nil {
			return reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Second}, err
		}

		// start delete process
		r.backgroundRunner(doDelete, machineInstance, "handleDelete")

	}

	return reconcile.Result{}, nil
}

func deleteMachine(r *ReconcileMachine, machine *clusterv1alpha1.CnctMachine) error {
	systemID := machine.ObjectMeta.Annotations["maas-system-id"]
	if systemID == "" {
		goto removeFinalizers
	}

	if err := r.MAASClient.Delete(context.Background(), &maas.DeleteRequest{SystemID: systemID}); err != nil {
		return errs.Wrapf(err, "could not delete machine %s with system id %q", machine.Name, *machine.Spec.ProviderID)
	}

removeFinalizers:
	machine.Finalizers = util.RemoveString(machine.Finalizers, clusterv1alpha1.MachineFinalizer)
	if err := r.updateStatus(machine, corev1.EventTypeNormal,
		common.ResourceStateChange, common.MessageResourceStateChange,
		machine.GetName(), common.DeletingMachinePhase); err != nil {
		klog.Errorf("failed to update status for machine %s: %q", machine.GetName(), err)
	}
	return nil
}

func (r *ReconcileMachine) handleUpgrade(machineInstance *clusterv1alpha1.CnctMachine, clusterInstance *clusterv1alpha1.CnctCluster) (reconcile.Result, error) {
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
		klog.Errorf("could not list Machines for cluster %s: %q", machineInstance.GetName(), err)
		return reconcile.Result{}, err
	}
	// if not ok to upgrade with error, return and do not requeue
	ok, err := util.IsReadyForUpgrade(machineList)
	if err != nil {
		klog.Errorf("cannot upgrade machine %s: %q", machineInstance.GetName(), err)
		return reconcile.Result{}, nil
	}
	// if not ok to upgrade, try later
	if !ok {
		klog.Infof("Upgrade: Waiting for cluster %s to finish reconciling", machineInstance.Spec.ClusterRef)
		return reconcile.Result{Requeue: true}, nil
	}

	// update status to "upgrading"
	machineInstance.Status.Phase = common.UpgradingMachinePhase
	err = r.updateStatus(machineInstance, corev1.EventTypeNormal,
		common.ResourceStateChange, common.MessageResourceStateChange,
		machineInstance.GetName(), common.UpgradingMachinePhase)
	if err != nil {
		klog.Errorf("could not update status of machine %s: %q", machineInstance.GetName(), err)
		return reconcile.Result{}, err
	}

	// otherwise start upgrade process
	r.backgroundRunner(doUpgrade, machineInstance, "handleUpgrade")

	return reconcile.Result{}, nil
}

func (r *ReconcileMachine) handleCreate(machine *clusterv1alpha1.CnctMachine, cluster *clusterv1alpha1.CnctCluster) (reconcile.Result, error) {
	// Add the finalizer
	if !util.ContainsString(machine.Finalizers, clusterv1alpha1.MachineFinalizer) {
		klog.Infoln("adding finalizer to machine")
		machine.Finalizers =
			append(machine.Finalizers, clusterv1alpha1.MachineFinalizer)
	}

	klog.Infoln("get cluster-private-key secret")
	secret := corev1.Secret{}
	if err := r.Client.Get(context.Background(), client.ObjectKey{Name: "cluster-private-key", Namespace: machine.Namespace}, &secret); err != nil {
		klog.Error("could not get cluster secret data")
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second}, err
	}
	klog.Infoln("create ca bundle from secret")
	bundle, err := cert.CABundleFromMap(secret.Data)
	if err != nil {
		return reconcile.Result{}, err
	}

	var userdata string
	isMaster := util.ContainsRole(machine.Spec.Roles, "master")
	if isMaster {
		klog.Infoln("write master userdata")
		userdata, err = masterUserdata(bundle)
	} else {
		klog.Infoln("get master machine")
		machineList, err := util.GetClusterMachineList(r.Client, cluster.GetName())
		if err != nil {
			return reconcile.Result{}, err
		}
		master, err := util.GetMaster(machineList)
		if err != nil {
			// No master found
			// TODO: set to error state
			return reconcile.Result{}, err
		}
		masterIP, ok := master.ObjectMeta.Annotations["maas-ip"]
		if !ok || masterIP == "" {
			return reconcile.Result{Requeue: true}, nil
		}
		apiserverAddress := fmt.Sprintf("%s:6443", masterIP)
		userdata, err = workerUserdata(bundle, apiserverAddress)
	}
	if err != nil {
		return reconcile.Result{}, err
	}

	// TODO: ProviderID should be unique. One way to ensure this is to generate
	// a UUID. Cf. k8s.io/apimachinery/pkg/util/uuid
	providerID := fmt.Sprintf("%s-%s", cluster.Name, machine.Name)
	createResponse, err := r.MAASClient.Create(context.Background(), &maas.CreateRequest{
		ProviderID: providerID,
		Distro:     "ubuntu-18.04-cnct-k8s-master",
		Userdata:   userdata})
	if err != nil {
		return reconcile.Result{}, err
	}

	if len(createResponse.IPAddresses) == 0 {
		klog.Info("Error machine (%s) ip is nil, releasing", providerID)
		r.MAASClient.Delete(context.Background(), &maas.DeleteRequest{ProviderID: createResponse.ProviderID, SystemID: createResponse.SystemID})
		return reconcile.Result{}, nil
	}

	if isMaster {
		klog.Info("create kubeconfig")
		kubeconfig, err := bundle.Kubeconfig(cluster.Name, "https://"+createResponse.IPAddresses[0]+":6443")
		if err != nil {
			return reconcile.Result{}, nil
		}

		klog.Info("add kubeconfig to cluster-private-key secret")
		secret.Data["kubeconfig"] = kubeconfig
		if err := r.Client.Update(context.Background(), &secret); err != nil {
			return reconcile.Result{}, err
		}
	}

	klog.Info("update machine status to ready")
	// update status to "creating"
	machine.Status.Phase = common.ReadyMachinePhase
	machine.Status.KubernetesVersion = cluster.Spec.KubernetesVersion
	machine.ObjectMeta.Annotations["maas-ip"] = createResponse.IPAddresses[0]
	machine.ObjectMeta.Annotations["maas-system-id"] = createResponse.SystemID
	err = r.updateStatus(machine, corev1.EventTypeNormal,
		common.ResourceStateChange, common.MessageResourceStateChange,
		machine.GetName(), common.ProvisioningMachinePhase)
	if err != nil {
		klog.Errorf("could not update status of machine %s: %q", machine.GetName(), err)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func masterUserdata(bundle *cert.CABundle) (string, error) {
	const userdataTmpl = `#cloud-config
write_files:
 - encoding: b64
   content: %s
   owner: root:root
   path: /etc/kubernetes/pki/certs.tar
   permissions: '0600'

runcmd:
 - [ sh, -c, "swapoff -a" ]
 - [ sh, -c, "tar xf /etc/kubernetes/pki/certs.tar -C /etc/kubernetes/pki" ]
 - [ sh, -c, "kubeadm init --pod-network-cidr 10.244.0.0/16 --token=andrew.isthebestfighter" ]
 - [ sh, -c, "kubectl --kubeconfig /etc/kubernetes/admin.conf apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml" ]

output : { all : '| tee -a /var/log/cloud-init-output.log' }
`
	caTar, err := bundle.ToTar()
	if err != nil {
		return "", err
	}
	userdata := fmt.Sprintf(userdataTmpl, caTar)
	return userdata, nil
}

func workerUserdata(bundle *cert.CABundle, apiserverAddress string) (string, error) {
	const userdataTmpl = `#cloud-config
runcmd:
 - [ sh, -c, "swapoff -a" ]
 - [ sh, -c, "kubeadm join --discovery-token=andrew.isthebestfighter --discovery-token-ca-cert-hash %s %s" ]

output : { all : '| tee -a /var/log/cloud-init-output.log' }
`
	certBlock, _ := pem.Decode(bundle.K8s)
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return "", errs.Wrap(err, "could not parse k8s certificate for public key")
	}
	hash := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
	caHash := fmt.Sprintf("sha256:%x", hash)
	userdata := fmt.Sprintf(userdataTmpl, caHash, apiserverAddress)
	return userdata, nil
}

func (r *ReconcileMachine) handleWaitingForReady(machine *clusterv1alpha1.CnctMachine, cluster *clusterv1alpha1.CnctCluster) (reconcile.Result, error) {
	klog.Infof("figure out how to check if a machine is ready")
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

	machineFreshInstance.ObjectMeta.Annotations = machineInstance.ObjectMeta.Annotations
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

func doUpgrade(r *ReconcileMachine, machineInstance *clusterv1alpha1.CnctMachine, privateKey []byte) error {
	cfg, err := NewCmdConfig(r.Client, machineInstance, privateKey)
	if err != nil {
		return err
	}

	// if master, do the upgrade.
	// otherwise wait for master to finish upgrade then
	// proceed with upgrade
	// master is done upgrading when it is in ready status and
	// its status kubernetes version matches clusters kubernetes version
	if util.ContainsRole(machineInstance.Spec.Roles, common.MachineRoleMaster) {
		klog.Infof("running upgrade on master %s for cluster %s",
			machineInstance.GetName(), machineInstance.Spec.ClusterRef)
		if err := UpgradeMaster(cfg, nil); err != nil {
			return err
		}
	} else {
		klog.Infof("running upgrade on worker %s for cluster %s",
			machineInstance.GetName(), machineInstance.Spec.ClusterRef)
		err = util.Retry(120, 10*time.Second, func() error {
			// get list of machines
			machineList, err := util.GetClusterMachineList(r.Client, cfg.clusterInstance.GetName())
			if err != nil {
				klog.Errorf("could not list Machines for cluster %s: %q",
					machineInstance.Spec.ClusterRef, err)
				return err
			}

			masterMachine, err := util.GetMaster(machineList)
			if err != nil {
				klog.Errorf("could not get master instance for cluster %s: %q",
					machineInstance.Spec.ClusterRef, err)
				return err
			}

			if masterMachine.Status.Phase != common.ReadyMachinePhase {
				klog.Infof("master %s of cluster %s is not ready, will retry machine %s", masterMachine.GetName(),
					machineInstance.Spec.ClusterRef, machineInstance.GetName())
				return fmt.Errorf("master %s of cluster %s is not ready", masterMachine.GetName(),
					machineInstance.Spec.ClusterRef)
			}

			if masterMachine.Status.KubernetesVersion != cfg.clusterInstance.Spec.KubernetesVersion {
				klog.Infof("master %s of cluster %s is not done with upgrade, will retry machine %s",
					masterMachine.GetName(), machineInstance.Spec.ClusterRef, machineInstance.GetName())
				return fmt.Errorf("master %s of cluster %s is not ready", masterMachine.GetName(),
					machineInstance.Spec.ClusterRef)
			}

			klog.Info("master phase: " + string(masterMachine.Status.Phase) +
				" master version: " + masterMachine.Status.KubernetesVersion)
			return nil
		})
		if err != nil {
			klog.Error(err, "Master failed to upgrade in time")
			return err
		}

		machineList, err := util.GetClusterMachineList(r.Client, cfg.clusterInstance.GetName())
		if err != nil {
			klog.Errorf("could not list Machines for cluster %s: %q",
				cfg.clusterInstance.GetName(), err)
			return err
		}

		masterMachine, err := util.GetMaster(machineList)
		if err != nil {
			klog.Errorf("could not get master instance for cluster %s: %q",
				cfg.clusterInstance.GetName(), err)
			return err
		}

		masterCfg := cfg
		masterCfg.machineInstance = masterMachine
		// get admin kubeconfig
		kubeConfig, err := GetKubeConfig(masterCfg, nil)
		if err != nil {
			return err
		}

		// run node upgrade
		err = UpgradeNode(cfg, map[string]string{"admin.conf": string(kubeConfig)})
		if err != nil {
			return err
		}
	}

	machineInstance.Status.Phase = common.ReadyMachinePhase
	machineInstance.Status.KubernetesVersion = cfg.clusterInstance.Spec.KubernetesVersion

	err = r.updateStatus(machineInstance, corev1.EventTypeNormal,
		common.ResourceStateChange, common.MessageResourceStateChange,
		machineInstance.GetName(), common.ReadyMachinePhase)
	if err != nil {
		return err
	}

	return nil
}

func doDelete(r *ReconcileMachine, machineInstance *clusterv1alpha1.CnctMachine, privateKey []byte) error {
	// there is a few possibilities here:
	// 1. Cluster is being deleted. We can just delete this machine
	// 2. Worker machine is being deleted. We should drain the machine first, and then delete it
	// 3. Master machine is being deleted. We should issue a warning, and then delete it.
	cfg, err := NewCmdConfig(r.Client, machineInstance, privateKey)
	if err != nil {
		return err
	}

	// Cluster is NOT being deleted.
	if cfg.clusterInstance.Status.Phase != common.StoppingClusterPhase {
		// get the master machine
		machineList, err := util.GetClusterMachineList(r.Client, cfg.clusterInstance.GetName())
		if err != nil {
			klog.Errorf("could not list Machines for cluster %s: %q",
				cfg.clusterInstance.GetName(), err)
			return errs.Wrapf(err, "could not list Machines for cluster %s", cfg.clusterInstance.GetName())
		}
		masterMachine, err := util.GetMaster(machineList)
		if err != nil {
			klog.Errorf("could not get master for cluster %s: %q",
				cfg.clusterInstance.GetName(), err)
			return errs.Wrapf(err, "could not get master for cluster %s", cfg.clusterInstance.GetName())
		}

		// TODO: this will need to be handled better
		if masterMachine.GetName() == machineInstance.GetName() {
			klog.Infof("WARNING!!! DELETING MASTER, %s"+
				"CLUSTER %s WILL NOT FUNCTION WITHOUT NEW MASTER AND FULL RESET",
				masterMachine.GetName(), cfg.clusterInstance.GetName())
		}

		masterCfg := cfg
		masterCfg.machineInstance = masterMachine
		// get admin kubeconfig
		kubeConfig, err := GetKubeConfig(masterCfg, nil)

		// run node drain
		args := map[string]string{"admin.conf": string(kubeConfig)}
		if err := DrainAndDeleteNode(cfg, args); err != nil {
			return err
		}
	}

	// run delete command
	if err := DeleteNode(cfg, nil); err != nil {
		return errs.Wrapf(
			err,
			"failed to clean up physical node for machine %s Manual cleanup might be required for %s",
			machineInstance.GetName(),
			machineInstance.Spec.SshConfig.Host,
		)
	}

	machineInstance.Finalizers =
		util.RemoveString(machineInstance.Finalizers, clusterv1alpha1.MachineFinalizer)
	if err := r.updateStatus(machineInstance, corev1.EventTypeNormal,
		common.ResourceStateChange, common.MessageResourceStateChange,
		machineInstance.GetName(), common.DeletingMachinePhase); err != nil {
		klog.Errorf("failed to update status for machine %s: %q", machineInstance.GetName(), err)
	}

	return err
}

func (r *ReconcileMachine) backgroundRunner(op backgroundMachineOp,
	machineInstance *clusterv1alpha1.CnctMachine, operationName string) {
	// TODO: add cancelable context to the commands so we don't continue running
	//  them after a timeout.
	type commandError struct {
		Err error
	}

	// start bootstrap command (or pre upgrade etc)
	opResult := make(chan commandError)
	timer := time.NewTimer(30 * time.Minute)

	// get the cluster instance
	clusterInstance, err := getCluster(r.Client, machineInstance.GetNamespace(), machineInstance.Spec.ClusterRef)
	if err != nil {
		klog.Errorf("Could not get cluster %q: %s", machineInstance.Spec.ClusterRef, err)
		return
	}

	privateKeySecret, err := k8sutil.GetSecret(r.Client, clusterInstance.Spec.Secret, clusterInstance.GetNamespace())
	if err != nil {
		klog.Errorf("Could not get cluster %s private key secret %s: %q",
			clusterInstance.GetName(), clusterInstance.Spec.Secret, err)
		return
	}
	privateKey := privateKeySecret.Data["private-key"]

	// TODO: op should just take a context.WithTimeout

	go func(ch chan<- commandError) {
		err := op(r, machineInstance, privateKey)
		ch <- commandError{Err: err}
		close(opResult)
	}(opResult)

	go func() {
		timedOut := false
		select {
		// cluster operation timeouts shouldn't take longer than 10 minutes
		case <-timer.C:
			err := fmt.Errorf("operation %s timed out for machine %s",
				operationName, machineInstance.GetNamespace())
			klog.Errorf("Provisioning operation timed out for object machine %s, cluster %s: %q",
				machineInstance.GetName(), machineInstance.Spec.ClusterRef, err)

			timer.Stop()
			timedOut = true
		case result := <-opResult:
			// if finished with error
			if result.Err != nil {
				switch err := errs.Cause(result.Err).(type) {
				case ssh.ErrorCmd:
					klog.Errorf(
						"could not complete machine %s pre-start step %s command %s: %q",
						machineInstance.GetName(),
						operationName,
						err.Cmd(),
						result.Err,
					)
				default:
					klog.Errorf(
						"could not complete machine %s pre-start: %q",
						machineInstance.GetName(),
						err,
					)
				}

				// set error status
				machineInstance.Status.Phase = common.ErrorMachinePhase
				err := r.updateStatus(machineInstance, corev1.EventTypeWarning,
					common.ResourceStateChange, common.MessageResourceStateChange,
					machineInstance.GetName(), common.ErrorMachinePhase)
				if err != nil {
					klog.Errorf(
						"could not update status of machine %s of cluster %s: %q",
						machineInstance.GetName(), machineInstance.Spec.ClusterRef,
						err,
					)
				}
			}
		}

		// if timed out, wait for operation to complete
		if timedOut {
			// TODO: fix this so we cancel the command. Currently this will
			//  always run until completion or failure.
			result := <-opResult
			if result.Err != nil {
				switch err := errs.Cause(result.Err).(type) {
				case ssh.ErrorCmd:
					klog.Errorf(
						"Timed out machine %s pre-start step %s command %s: %q",
						machineInstance.GetName(),
						operationName,
						err.Cmd(),
						err.Error(),
					)
				}
			} else {
				klog.Errorf(
					"Timed out machine %s but completed \"successfully\".",
					machineInstance.GetName(),
				)
			}

			// TODO: this sets the machine to ErrorMachinePhase even if it
			//  eventually completed successfully.
			// set error status
			machineInstance.Status.Phase = common.ErrorMachinePhase
			err := r.updateStatus(machineInstance, corev1.EventTypeWarning,
				common.ResourceStateChange, common.MessageResourceStateChange,
				machineInstance.GetName(), common.ErrorMachinePhase)
			if err != nil {
				klog.Errorf("could not update status of machine %s cluster %s: %q",
					machineInstance.GetName(), machineInstance.Spec.ClusterRef, err)
			}
		}
	}()
}
