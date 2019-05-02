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

package v1alpha1

import (
	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MachineSetSpec defines the desired state of CnctMachineSet
type MachineSetSpec struct {
	// Replicas defines the number of type Machine
	Replicas int `json:"replicas,omitempty"`

	// MachineTemplate defines the desired state of each instance of Machine
	MachineTemplate MachineTemplate `json:"machineTemplate,omitempty"`
}

type MachineTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MachineSpec `json:"spec,omitempty"`
}

// MachineSetStatus defines the observed state of MachineSet
type MachineSetStatus struct {
	// When was this status last observed
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`

	// MachineSet status
	Phase common.MachineSetStatusPhase `json:"phase,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CnctMachineSet is the Schema for the cnctmachinesets API
// +k8s:openapi-gen=true
type CnctMachineSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineSetSpec   `json:"spec,omitempty"`
	Status MachineSetStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachineSetList contains a list of CnctMachineSet
type CnctMachineSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CnctMachineSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CnctMachineSet{}, &CnctMachineSetList{})
}
