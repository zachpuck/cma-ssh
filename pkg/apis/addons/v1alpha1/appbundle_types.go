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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AppBundleStatusPhase string

const (
	InstalledAppBundlePhase AppBundleStatusPhase = "InstalledAppBundle"
)

// AppBundleSpec defines the desired state of AppBundle
type AppBundleSpec struct {
	// Image used for install (including tag)
	Image string `json:"image"`

	// Command run in image (ex: helm install stable/nginx-ingress)
	Command string `json:"command,omitempty"`
}

// AppBundleStatus defines the observed state of AppBundle
type AppBundleStatus struct {
	// AppBundle status
	Phase AppBundleStatusPhase `json:"phase,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AppBundle is the Schema for the appbundles API
// +k8s:openapi-gen=true
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="appbundle status"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type AppBundle struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppBundleSpec   `json:"spec,omitempty"`
	Status AppBundleStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AppBundleList contains a list of AppBundle
type AppBundleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AppBundle `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AppBundle{}, &AppBundleList{})
}
