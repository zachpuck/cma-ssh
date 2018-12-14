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
