package machine

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"strings"
	"text/template"
	"time"

	"github.com/pkg/errors"
	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/common"
	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
	"github.com/samsung-cnct/cma-ssh/pkg/cert"
	"github.com/samsung-cnct/cma-ssh/pkg/maas"
	"github.com/samsung-cnct/cma-ssh/pkg/util"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeSchema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type errNotReady string

func (e errNotReady) Error() string {
	return string(e)
}

type errRelease struct {
	systemID string
	err      error
}

func (e errRelease) Error() string {
	return e.err.Error()
}

type clientEventer interface {
	client.Client
	record.EventRecorder
}

type creator struct {
	k8sClient  clientEventer
	maasClient maas.Client
	machine    *v1alpha1.CnctMachine
	err        error

	// derived types
	isMaster       bool
	cluster        v1alpha1.CnctCluster
	clientset      *kubernetes.Clientset
	secret         corev1.Secret
	token          string
	createRequest  maas.CreateRequest
	createResponse maas.CreateResponse
}

func create(k8sClient clientEventer, maasClient maas.Client, machine *v1alpha1.CnctMachine) error {
	klog.Info("checking if machine is master")
	var isMaster bool
	for _, v := range machine.Spec.Roles {
		if v == common.MachineRoleMaster {
			isMaster = true
			break
		}
	}
	c := &creator{k8sClient: k8sClient, maasClient: maasClient, machine: machine}
	c.isMaster = isMaster
	if isMaster {
		c.getCluster()
		c.getSecret()
		c.prepareMaasRequest()
		c.doMaasCreate()
		c.createKubeconfig()
		c.updateCluster()
		c.updateMachine()
	} else {
		c.getCluster()
		c.getSecret()
		c.createClientsetFromSecret()
		c.checkIfTokenExists()
		c.createToken()
		c.checkApiserverAddress()
		c.prepareMaasRequest()
		c.doMaasCreate()
		c.updateMachine()
	}
	return c.err
}

func (c *creator) getCluster() {
	if c.err != nil {
		return
	}

	klog.Info("getting cluster from namespace")
	// Get cluster from machine's namespace.
	var clusters v1alpha1.CnctClusterList
	c.err = c.k8sClient.List(
		context.Background(),
		&client.ListOptions{Namespace: c.machine.Namespace},
		&clusters,
	)
	if c.err != nil {
		return
	}
	if len(clusters.Items) == 0 {
		klog.Infof("no cluster in namespace, requeue request")
		c.err = errNotReady("no cluster in namespace")
		return
	}
	c.cluster = clusters.Items[0]
}

func (c *creator) getSecret() {
	if c.err != nil {
		return
	}
	klog.Info("getting the secret cert bundle")
	c.err = c.k8sClient.Get(context.Background(), client.ObjectKey{Name: "cluster-private-key", Namespace: c.machine.Namespace}, &c.secret)
}

func (c *creator) createClientsetFromSecret() {
	if c.err != nil || c.isMaster {
		return
	}

	klog.Info("creating clientset from cert bundle")
	configData, ok := c.secret.Data["kubeconfig"]
	if !ok || len(configData) == 0 {
		c.err = errNotReady("no kubeconfig in secret")
		return
	}
	config, err := clientcmd.NewClientConfigFromBytes(configData)
	if err != nil {
		c.err = err
		return
	}
	restConfig, err := config.ClientConfig()
	if err != nil {
		c.err = err
		return
	}
	c.clientset, c.err = kubernetes.NewForConfig(restConfig)
}

func (c *creator) checkIfTokenExists() {
	if c.err != nil || c.isMaster {
		return
	}
	klog.Info("checking for existing tokens on managed cluster")
	list, err := c.clientset.CoreV1().
		Secrets(metav1.NamespaceSystem).
		List(metav1.ListOptions{FieldSelector: "type=" + string(corev1.SecretTypeBootstrapToken)})
	if err != nil {
		e, ok := err.(*apierrors.StatusError)
		if !ok {
			c.err = errNotReady(err.Error())
		} else {
			c.err = e
		}
		return
	}

	// find the first non-expired token
	for _, secret := range list.Items {
		expires, ok := secret.Data["expiration"]
		if ok && len(expires) > 0 {
			t, err := time.Parse(time.RFC3339, string(expires))
			if err != nil || t.Before(time.Now()) {
				continue
			}
			klog.Info("found an existing token")
			c.token = fmt.Sprintf("%s.%s", list.Items[0].Data["token-id"], list.Items[0].Data["token-secret"])
			return
		}
		klog.Info("found an existing token")
		c.token = fmt.Sprintf("%s.%s", list.Items[0].Data["token-id"], list.Items[0].Data["token-secret"])
		return
	}
}

func (c *creator) createToken() {
	if c.err != nil || c.isMaster || c.token != "" {
		return
	}

	klog.Info("creating join token")
	tokBuf := make([]byte, 3)
	_, c.err = io.ReadFull(rand.Reader, tokBuf)
	if c.err != nil {
		return
	}
	tokSecBuf := make([]byte, 8)
	_, c.err = io.ReadFull(rand.Reader, tokSecBuf)
	if c.err != nil {
		return
	}
	c.err = createBootstrapToken(c.clientset, fmt.Sprintf("%x", tokBuf), fmt.Sprintf("%x", tokSecBuf))
	if c.err != nil {
		return
	}
	c.token = fmt.Sprintf("%x.%x", tokBuf, tokSecBuf)
}

func createBootstrapToken(clientset *kubernetes.Clientset, tokenID, tokenSecret string) error {
	token := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bootstrap-token-" + tokenID,
			Namespace: "kube-system",
		},
		Type: corev1.SecretTypeBootstrapToken,
		Data: map[string][]byte{
			"description":                    []byte("Bootstrap token created by cma-ssh"),
			"token-id":                       []byte(tokenID),
			"token-secret":                   []byte(tokenSecret),
			"expiration":                     []byte(time.Now().Add(1 * time.Hour).Format(time.RFC3339)),
			"usage-bootstrap-authentication": []byte("true"),
			"usage-bootstrap-signing":        []byte("true"),
			"auth-extra-groups":              []byte("system:bootstrappers:kubeadm:default-node-token"),
		},
	}
	if _, err := clientset.CoreV1().Secrets("kube-system").Create(&token); err != nil {
		return errors.Wrap(err, "could not create token secret")
	}
	return nil
}

func (c *creator) checkApiserverAddress() {
	if c.err != nil || c.isMaster {
		return
	}

	klog.Info("checking if apiendpoint is set on cluster")
	apiserverAddress := c.cluster.Status.APIEndpoint
	if apiserverAddress == "" {
		c.err = errNotReady(fmt.Sprintf("%s cluster APIEndpoint is not set", c.cluster.Name))
		return
	}
}

func (c *creator) prepareMaasRequest() {
	if c.err != nil {
		return
	}

	klog.Infoln("preparing maas request")
	var bundle *cert.CABundle
	bundle, c.err = cert.CABundleFromMap(c.secret.Data)
	if c.err != nil {
		return
	}
	var userdata string
	if c.isMaster {
		userdata, c.err = masterUserdata(bundle)
	} else {
		userdata, c.err = workerUserdata(bundle, c.token, c.cluster.Status.APIEndpoint)
	}
	// TODO: ProviderID should be unique. One way to ensure this is to generate
	// a UUID. Cf. k8s.io/apimachinery/pkg/util/uuid
	providerID := fmt.Sprintf("%s-%s", c.cluster.Name, c.machine.Name)
	c.createRequest = maas.CreateRequest{
		ProviderID:   providerID,
		Distro:       "ubuntu-18.04-cnct-k8s-master",
		Userdata:     userdata,
		InstanceType: c.machine.Spec.InstanceType,
	}
}

const masterUserdataTmplText = `#cloud-config
write_files:
 - encoding: b64
   content: {{ .Tar }}
   owner: root:root
   path: /etc/kubernetes/pki/certs.tar
   permissions: '0600'

runcmd:
 - [ sh, -c, "swapoff -a" ]
 - [ sh, -c, "tar xf /etc/kubernetes/pki/certs.tar -C /etc/kubernetes/pki" ]
 - [ sh, -c, "kubeadm init --pod-network-cidr 10.244.0.0/16" ]
 - [ sh, -c, "kubectl --kubeconfig /etc/kubernetes/admin.conf apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml" ]

output : { all : '| tee -a /var/log/cloud-init-output.log' }
`

var masterUserdataTmpl = template.Must(template.New("master").Parse(masterUserdataTmplText))

func masterUserdata(bundle *cert.CABundle) (string, error) {
	caTar, err := bundle.ToTar()
	if err != nil {
		return "", err
	}
	var userdata strings.Builder
	data := struct {
		Tar string
	}{
		Tar: caTar,
	}
	if err := masterUserdataTmpl.Execute(&userdata, data); err != nil {
		return "", err
	}
	return userdata.String(), nil
}

const workerUserdataTmplText = `#cloud-config
runcmd:
 - [ sh, -c, "swapoff -a" ]
 - [ sh, -c, "kubeadm join --discovery-token={{ .Token }} --discovery-token-ca-cert-hash {{ .CertHash }} {{ .APIEndpoint }}" ]

output : { all : '| tee -a /var/log/cloud-init-output.log' }
`

var workerUserdataTmpl = template.Must(template.New("worker").Parse(workerUserdataTmplText))

func workerUserdata(bundle *cert.CABundle, token, apiserverAddress string) (string, error) {
	certBlock, _ := pem.Decode(bundle.K8s)
	certificate, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return "", errors.Wrap(err, "could not parse k8s certificate for public key")
	}
	hash := sha256.Sum256(certificate.RawSubjectPublicKeyInfo)
	caHash := fmt.Sprintf("sha256:%x", hash)
	var buf strings.Builder
	data := struct{ Token, CertHash, APIEndpoint string }{token, caHash, apiserverAddress}
	if err := workerUserdataTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (c *creator) doMaasCreate() {
	if c.err != nil {
		return
	}

	klog.Info("calling create on maas")
	createResponse, err := c.maasClient.Create(context.Background(), &c.createRequest)
	if err != nil {
		c.err = err
		return
	}

	if len(createResponse.IPAddresses) == 0 {
		klog.Infof("Error machine (%s) ip is nil, releasing", createResponse.ProviderID)
		c.err = c.maasClient.Delete(context.Background(), &maas.DeleteRequest{ProviderID: createResponse.ProviderID, SystemID: createResponse.SystemID})
		return
	}
	c.createResponse = *createResponse
}

func (c *creator) createKubeconfig() {
	if c.err != nil || !c.isMaster {
		return
	}

	klog.Info("creating kubeconfig")
	bundle, err := cert.CABundleFromMap(c.secret.Data)
	if err != nil {
		c.err = err
		return
	}

	klog.Info("create kubeconfig")
	kubeconfig, err := bundle.Kubeconfig(c.cluster.Name, "https://"+c.createResponse.IPAddresses[0]+":6443")
	if err != nil {
		c.err = err
		return
	}

	klog.Info("add kubeconfig to cluster-private-key secret")
	c.secret.Data["kubeconfig"] = kubeconfig
	c.err = c.k8sClient.Update(context.Background(), &c.secret)
}

func (c *creator) updateMachine() {
	if c.err != nil {
		return
	}

	klog.Info("updating machine")
	freshMachine := v1alpha1.CnctMachine{}
	c.err = c.k8sClient.Get(
		context.Background(),
		client.ObjectKey{
			Namespace: c.machine.GetNamespace(),
			Name:      c.machine.GetName(),
		},
		&freshMachine,
	)
	if c.err != nil {
		return
	}

	// Add the finalizer
	if !util.ContainsString(freshMachine.Finalizers, v1alpha1.MachineFinalizer) {
		klog.Infoln("adding finalizer to machine")
		freshMachine.Finalizers = append(freshMachine.Finalizers, v1alpha1.MachineFinalizer)
	}

	// Set owner ref
	machineOwnerRef := *metav1.NewControllerRef(&c.cluster,
		runtimeSchema.GroupVersionKind{
			Group:   v1alpha1.SchemeGroupVersion.Group,
			Version: v1alpha1.SchemeGroupVersion.Version,
			Kind:    "CnctCluster",
		},
	)

	freshMachine.OwnerReferences = append(freshMachine.OwnerReferences, machineOwnerRef)

	klog.Info("update machine status to ready")
	// update status to "creating"
	freshMachine.Status.Phase = common.ReadyMachinePhase
	freshMachine.Status.KubernetesVersion = c.cluster.Spec.KubernetesVersion
	freshMachine.ObjectMeta.Annotations["maas-ip"] = c.createResponse.IPAddresses[0]
	freshMachine.ObjectMeta.Annotations["maas-system-id"] = c.createResponse.SystemID

	c.err = c.k8sClient.Update(context.Background(), &freshMachine)
	if c.err != nil {
		return
	}

	c.k8sClient.Event(
		&freshMachine,
		corev1.EventTypeNormal,
		"ResourceStateChange",
		"set Finalizer and OwnerReferences",
	)
}

func (c *creator) updateCluster() {
	if c.err != nil || !c.isMaster {
		return
	}

	klog.Info("updating cluster")
	klog.Info("updating cluster api endpoint")
	var fresh v1alpha1.CnctCluster
	err := c.k8sClient.Get(
		context.Background(),
		client.ObjectKey{
			Namespace: c.cluster.GetNamespace(),
			Name:      c.cluster.GetName(),
		},
		&fresh,
	)
	if err != nil {
		c.err = errRelease{err: err, systemID: c.createResponse.SystemID}
	}

	fresh.Status.APIEndpoint = c.createResponse.IPAddresses[0] + ":6443"
	fresh.Status.LastUpdated = &metav1.Time{Time: time.Now()}
	err = c.k8sClient.Update(context.Background(), &fresh)
	if err != nil {
		c.err = errRelease{err: err, systemID: c.createResponse.SystemID}
		return
	}

	c.k8sClient.Event(
		&fresh,
		corev1.EventTypeNormal,
		"ResourceStateChange",
		"set cluster APIEndpoint",
	)
}
