package k8sutil

import (
	"context"
	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func GetSecretList(c client.Client, options *client.ListOptions) ([]corev1.Secret, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("k8sutil secrets GetSecretList()")

	secrets := &corev1.SecretList{}
	err := c.List(
		context.Background(),
		options,
		secrets)
	if err != nil {
		log.Error(err, "failed to list secrets in Namespace: ", options.Namespace)
		return nil, err
	}

	return secrets.Items, nil
}

func DeleteSecret(c client.Client, name string, namespace string) error {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("k8sutil secrets DeleteSecret()")

	secret := &corev1.Secret{}
	err := c.Delete(
		context.Background(),
		secret)
	if err != nil {
		log.Error(err, "failed to delete Secret: ", name, " in Namespace: ", namespace)
		return err
	}

	return nil
}

func GetSecret(c client.Client, name string, namespace string) (corev1.Secret, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("k8sutil secrets GetSecret()")

	secret := &corev1.Secret{}
	err := c.Get(
		context.Background(),
		client.ObjectKey{Namespace: namespace, Name: name},
		secret)
	if err != nil {
		log.Error(err, "failed to get secret", "secret", name)
		return *secret, err
	}

	return *secret, err
}

func CreateSecret(c client.Client, secret *corev1.Secret) error {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("k8sutil secrets CreateSecret()")

	err := c.Create(
		context.Background(),
		secret)
	if err != nil {
		log.Error(err, "failed to create Secret: ", secret.Name)
		return err
	}

	return nil
}

func GetSSHSecret(c client.Client, name string, namespace string) ([]byte, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("k8sutil secrets GetSSHSecret()")

	secretResult, err := GetSecret(c, name, namespace)
	if secretResult.Type != corev1.SecretTypeOpaque {
		log.Error(err, "secret is not of type ", corev1.SecretTypeOpaque, ", but rather is of type -->", secretResult.Type, "<--")
		return nil, err
	}
	key := secretResult.Data["private-key"]
	return key, nil
}

func GetKubeconfigSecretList(c client.Client, namespace string) (result []corev1.Secret, err error) {

	listOption := client.ListOptions{
		Namespace: namespace,
	}

	listOption.SetFieldSelector("type=" + string(corev1.SecretTypeOpaque))
	listOption.SetLabelSelector("kubeconfig=true")
	return GetSecretList(c, &listOption)
}

func GetKubeconfigSecret(c client.Client, name string, namespace string) ([]byte, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("k8sutil secrets GetKubeconfigSecret()")

	secretResult, err := GetSecret(c, name, namespace)
	if err != nil {
		log.Error(err, "failed to get kubeconfig secret for cluster in ", "namespace", namespace)
		return nil, err
	}
	if secretResult.Type != corev1.SecretTypeOpaque {
		log.Error(err, "secret is not of ", "type", corev1.SecretTypeOpaque)
		return nil, err
	}
	if secretResult.Labels["kubeconfig"] != "true" {
		log.Error(err, "secret does not have ", "label", "kubeconfig=true")
		return nil, err
	}
	secret := secretResult.Data[corev1.ServiceAccountKubeconfigKey]
	return secret, nil
}

func DeleteKubeconfigSecret(c client.Client, name string, namespace string) error {
	return DeleteSecret(c, name, namespace)
}

func CreateKubeconfigSecret(c client.Client, clusterInstance *clusterv1alpha1.CnctCluster,
	scheme *runtime.Scheme, kubeconfig []byte) error {

	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("k8sutil secrets CreateKubeconfigSecret()")

	dataMap := make(map[string][]byte)
	dataMap[corev1.ServiceAccountKubeconfigKey] = kubeconfig

	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      clusterInstance.GetName() + "-kubeconfig",
			Namespace: clusterInstance.GetNamespace(),
		},
		Type: corev1.SecretTypeOpaque,
		Data: dataMap,
	}

	err := controllerutil.SetControllerReference(clusterInstance, secret, scheme)
	if err != nil {
		log.Error(err, "failed to set controller reference on secret",
			"controller", clusterInstance,
			"secret", secret)
		return err
	}

	err = CreateSecret(c, secret)
	if err != nil {
		log.Error(err, "failed to create kubeconfig secret",
			"secret", secret)
		return err
	}

	return nil
}
