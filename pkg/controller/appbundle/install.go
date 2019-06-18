package appbundle

import (
	"fmt"
	addonsv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/addons/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	CMAInstallNamespace      = "cma-install"
	CMAInstallServiceAccount = "cma-sa"
)

// create namespace on remote cluster to use for installing software
func createNamespace(clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().Namespaces().Get(CMAInstallNamespace, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: CMAInstallNamespace,
			},
		}
		_, err = clientset.CoreV1().Namespaces().Create(ns)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

// Create service account on remote cluster to use for installing software
func createServiceAccount(clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().ServiceAccounts(CMAInstallNamespace).Get(CMAInstallServiceAccount, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: CMAInstallServiceAccount,
			},
		}
		_, err = clientset.CoreV1().ServiceAccounts(CMAInstallNamespace).Create(sa)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

// create cluster role binding with cluster-admin role on remote cluster to use for installing software
func createClusterRoleBinding(clientset *kubernetes.Clientset) error {
	_, err := clientset.RbacV1().ClusterRoleBindings().Get(CMAInstallServiceAccount, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		crb := &rbac.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: CMAInstallServiceAccount,
			},
			Subjects: []rbac.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      CMAInstallServiceAccount,
					Namespace: CMAInstallNamespace,
				},
			},
			RoleRef: rbac.RoleRef{
				Kind:     "ClusterRole",
				Name:     "cluster-admin",
				APIGroup: "",
			},
		}
		_, err = clientset.RbacV1().ClusterRoleBindings().Create(crb)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

// create kubernetes job on remote cluster to use for installing software
func createInstallJob(clientset *kubernetes.Clientset, appBundle *addonsv1alpha1.AppBundle) error {
	jobName := fmt.Sprintf("%s-%s", appBundle.Name, "installer")
	_, err := clientset.BatchV1().Jobs(CMAInstallNamespace).Get(jobName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		container := corev1.Container{
			Name:            jobName,
			Image:           appBundle.Spec.Image,
			ImagePullPolicy: "Always",
		}
		if appBundle.Spec.Command != "" {
			container.Command = []string{appBundle.Spec.Command}
		}
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name: jobName,
			},
			Spec: batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						ServiceAccountName: CMAInstallServiceAccount,
						Containers:         []corev1.Container{container},
						RestartPolicy:      corev1.RestartPolicyOnFailure,
					},
				},
			},
		}
		_, err = clientset.BatchV1().Jobs(CMAInstallNamespace).Create(job)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}
