/*
Copyright 2018 Samsung SDS.
Copyright 2018 The Kubernetes Authors.

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

package v1alpha1

import (
	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const MachineFinalizer = "machine.cluster.sds.samsung.com"

// MachineSpec defines the desired state of Machine
type MachineSpec struct {
	// name of the Cluster object this node belongs to
	ClusterName string `json:"clustername,omitempty"`

	// The full, authoritative list of taints to apply to the corresponding
	// Node.
	// +optional
	Taints []corev1.Taint `json:"taints,omitempty"`

	Roles []common.MachineRoles `json:"roles,omitempty"`

	SshConfig MachineSshConfigInfo `json:"sshconfig,omitempty"`
}

// MachineSshConfigInfo defines the ssh configuration for the physical
// node represented by this Machine
type MachineSshConfigInfo struct {
	Username string `json:"username,omitempty"`

	Host string `json:"host,omitempty"`

	Port uint32 `json:"port,omitempty"`

	Secret string `json:"secret,omitempty"`
}

// MachineStatus defines the observed state of Machine
type MachineStatus struct {
	// When was this status last observed
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`

	// Machine status
	Phase common.MachineStatusPhase `json:"phase,omitempty"`

	// kubernetes version of the node, should be equal to corresponding cluster version
	KubernetesVersion string `json:"kubernetesVersion"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Machine is the Schema for the machines API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Machine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineSpec   `json:"spec,omitempty"`
	Status MachineStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachineList contains a list of Machine
type MachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Machine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Machine{}, &MachineList{})
}
