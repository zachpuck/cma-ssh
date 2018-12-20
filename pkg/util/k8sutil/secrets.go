package k8sutil

import (
	"context"
	"github.com/golang/glog"
	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func GetSecretList(c client.Client, options *client.ListOptions) ([]corev1.Secret, error) {
	secrets := &corev1.SecretList{}
	err := c.List(
		context.Background(),
		options,
		secrets)
	if err != nil {
		glog.Errorf("failed to list secrets in Namespace %s: %q", options.Namespace, err)
		return nil, err
	}

	return secrets.Items, nil
}

func DeleteSecret(c client.Client, name string, namespace string) error {
	secret := &corev1.Secret{}
	err := c.Delete(
		context.Background(),
		secret)
	if err != nil {
		glog.Errorf("failed to delete Secret %s in namespace %s: %q", name, namespace, err)
		return err
	}

	return nil
}

func GetSecret(c client.Client, name string, namespace string) (corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := c.Get(
		context.Background(),
		client.ObjectKey{Namespace: namespace, Name: name},
		secret)
	if err != nil {
		glog.Errorf("failed to get secret %s: %q", name, err)
		return *secret, err
	}

	return *secret, err
}

func CreateSecret(c client.Client, secret *corev1.Secret) error {
	err := c.Create(
		context.Background(),
		secret)
	if err != nil {
		glog.Errorf("failed to create Secret %s: %q", secret.Name, err)
		return err
	}

	return nil
}

func SetSecretOwner(c client.Client, secret *corev1.Secret,
	clusterInstance *clusterv1alpha1.CnctCluster, scheme *runtime.Scheme) error {
	err := controllerutil.SetControllerReference(clusterInstance, secret, scheme)
	if err != nil {
		glog.Errorf("failed to set controller %s reference on secret %s: %q",
			clusterInstance.GetName(), secret.Name, err)
		return err
	}

	err = c.Update(
		context.Background(),
		secret)
	if err != nil {
		glog.Errorf("failed to update Secret %s: %q", secret.Name, err)
		return err
	}

	return nil
}

func CreateKubeconfigSecret(c client.Client, clusterInstance *clusterv1alpha1.CnctCluster,
	scheme *runtime.Scheme, kubeconfig []byte) error {
	dataMap := make(map[string][]byte)
	dataMap[corev1.ServiceAccountKubeconfigKey] = kubeconfig

	secret, err := GetSecret(c, clusterInstance.GetName()+"-kubeconfig", clusterInstance.GetNamespace())
	if err != nil {
		if errors.IsNotFound(err) {
			newSecret := &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      clusterInstance.GetName() + "-kubeconfig",
					Namespace: clusterInstance.GetNamespace(),
				},
				Type: corev1.SecretTypeOpaque,
				Data: dataMap,
			}
			err = controllerutil.SetControllerReference(clusterInstance, newSecret, scheme)
			if err != nil {
				glog.Errorf("failed to set controller %s reference on secret %s: %q",
					clusterInstance.GetName(), secret.GetName(), err)
				return err
			}
			err = CreateSecret(c, newSecret)
			if err != nil {
				glog.Errorf("failed to create kubeconfig secret %s: %q", secret.GetName(), err)
				return err
			}
		} else {
			glog.Errorf("failed to query for secret %s: %q", clusterInstance.GetName()+"-kubeconfig", err)
			return err
		}
	} else {
		if !reflect.DeepEqual(secret.Data, dataMap) {
			secret.Data = dataMap
			err = c.Update(context.Background(), &secret)
			if err != nil {
				glog.Errorf("failed to update secret %s: %q", clusterInstance.GetName()+"-kubeconfig", err)
				return err
			}
		}
	}

	return nil
}
