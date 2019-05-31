package prometheus

import (
	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	CMA_Install_Namespace      = "cma-install"
	CMA_Install_ServiceAccount = "cma-sa"
	Prometheus_Installer_Image = "zachpuck/protest:latest"
)

func InstallPrometheus(clientset *kubernetes.Clientset) error {
	// create namespace to use for installing software on cluster
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: CMA_Install_Namespace,
		},
	}
	_, nsErr := clientset.CoreV1().Namespaces().Create(ns)
	if nsErr != nil {
		return errors.Wrap(nsErr, "failed to create namespace for cma")
	}

	// Create service account to use for installing software on cluster
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: CMA_Install_ServiceAccount,
		},
	}
	_, saErr := clientset.CoreV1().ServiceAccounts(CMA_Install_Namespace).Create(sa)
	if saErr != nil {
		return errors.Wrap(saErr, "failed to create service account for cma")
	}
	// create cluster role binding with cluster-admin role to use for installing software on cluster
	crb := &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: CMA_Install_ServiceAccount,
		},
		Subjects: []rbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      CMA_Install_ServiceAccount,
				Namespace: CMA_Install_Namespace,
			},
		},
		RoleRef: rbac.RoleRef{
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
			APIGroup: "",
		},
	}
	_, crbErr := clientset.RbacV1().ClusterRoleBindings().Create(crb)
	if crbErr != nil {
		return errors.Wrap(crbErr, "failed to create cluster role binding for cma")
	}

	// create job to install prometheus
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: "prometheus-installer",
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					ServiceAccountName: CMA_Install_ServiceAccount,
					Containers: []corev1.Container{
						{
							Name:            "prometheus-installer",
							Image:           Prometheus_Installer_Image,
							ImagePullPolicy: "Always",
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}
	_, proErr := clientset.BatchV1().Jobs(CMA_Install_Namespace).Create(job)
	if proErr != nil {
		return errors.Wrap(proErr, "failed to create prometheus Job")
	}
	return nil
}
