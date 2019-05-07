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

	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/common"
	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
	"github.com/samsung-cnct/cma-ssh/pkg/cert"
	"github.com/samsung-cnct/cma-ssh/pkg/maas"
	"github.com/samsung-cnct/cma-ssh/pkg/util"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeSchema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

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
		if apierrors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// log.Error(err, "could not find machine", "machine", request)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		klog.Errorf("could not read machine %s: %q", request.Name, err)
		return reconcile.Result{}, err
	}

	if !machine.DeletionTimestamp.IsZero() {
		return r.handleDelete(&machine)
	}

	switch machine.Status.Phase {
	case common.ProvisioningMachinePhase:
		return r.handleWaitingForReady(&machine)
	case common.DeletingMachinePhase:
		return r.handleDelete(&machine)
	case common.ErrorMachinePhase:
		return reconcile.Result{}, nil
	case common.ReadyMachinePhase:
		return reconcile.Result{}, nil
	case "":
		return r.handleCreate(&machine)
	default:
		klog.Errorf("unknown phase %q", machine.Status.Phase)
		return reconcile.Result{}, nil
	}
}

func (r *ReconcileMachine) handleDelete(machineInstance *clusterv1alpha1.CnctMachine) (reconcile.Result, error) {
	// get cluster status to determine whether we should proceed,
	// i.e. if there is a create in progress, we wait for it to either finish or error
	clusterName := util.GetClusterNameFromMachineOwnerRef(machineInstance)
	clusterInstance, err := getCluster(r.Client, machineInstance.GetNamespace(), clusterName)
	if err != nil {
		return reconcile.Result{}, err
	}
	machineList, err := util.GetClusterMachineList(r.Client, clusterInstance.GetName())
	if err != nil {
		err := errors.Wrap(err, "could not list Machines")
		klog.Error(err)
		return reconcile.Result{}, err
	}
	if !util.IsReadyForDeletion(machineList) {
		klog.Infof("Delete: Waiting for cluster %s to finish reconciling", clusterName)
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
	}

	return reconcile.Result{}, nil
}

func deleteMachine(r *ReconcileMachine, machine *clusterv1alpha1.CnctMachine) error {
	systemID := machine.ObjectMeta.Annotations["maas-system-id"]
	if systemID == "" {
		goto removeFinalizers
	}

	if err := r.MAASClient.Delete(context.Background(), &maas.DeleteRequest{SystemID: systemID}); err != nil {
		return errors.Wrapf(err, "could not delete machine %s with system id %q", machine.Name, *machine.Spec.ProviderID)
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
	clusterName := util.GetClusterNameFromMachineOwnerRef(machineInstance)
	clusterInstance, err := getCluster(r.Client, machineInstance.GetNamespace(), clusterName)
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
		klog.Infof("Upgrade: Waiting for cluster %s to finish reconciling", clusterName)
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

	return reconcile.Result{}, nil
}

func (r *ReconcileMachine) handleCreate(machine *clusterv1alpha1.CnctMachine) (reconcile.Result, error) {
	// Get cluster from machine's namespace.
	var clusters clusterv1alpha1.CnctClusterList
	err := r.List(
		context.Background(),
		&client.ListOptions{Namespace: machine.Namespace},
		&clusters,
	)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Either has not been created yet or has been deleted
			klog.Errorf("could not find cluster: %q", err)
			return reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Second}, nil
		}
		// Error reading the object - requeue the request.
		klog.Errorf("error reading object machine %s: %q", machine.GetName(), err)
		return reconcile.Result{}, err
	}
	if len(clusters.Items) == 0 {
		klog.Infof("no cluster in namespace, requeue request")
		return reconcile.Result{RequeueAfter: 5 * time.Second, Requeue: true}, nil
	}
	cluster := clusters.Items[0]

	// Add the finalizer
	if !util.ContainsString(machine.Finalizers, clusterv1alpha1.MachineFinalizer) {
		klog.Infoln("adding finalizer to machine")
		machine.Finalizers =
			append(machine.Finalizers, clusterv1alpha1.MachineFinalizer)
	}

	// Set owner ref
	machineOwnerRef := []metav1.OwnerReference{
		*metav1.NewControllerRef(&cluster,
			runtimeSchema.GroupVersionKind{
				Group:   clusterv1alpha1.SchemeGroupVersion.Group,
				Version: clusterv1alpha1.SchemeGroupVersion.Version,
				Kind:    "CnctCluster",
			},
		),
	}
	machine.OwnerReferences = machineOwnerRef

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
		klog.Infof("Error machine (%s) ip is nil, releasing", providerID)
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
	certificate, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return "", errors.Wrap(err, "could not parse k8s certificate for public key")
	}
	hash := sha256.Sum256(certificate.RawSubjectPublicKeyInfo)
	caHash := fmt.Sprintf("sha256:%x", hash)
	userdata := fmt.Sprintf(userdataTmpl, caHash, apiserverAddress)
	return userdata, nil
}

func (r *ReconcileMachine) handleWaitingForReady(machine *clusterv1alpha1.CnctMachine) (reconcile.Result, error) {
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
