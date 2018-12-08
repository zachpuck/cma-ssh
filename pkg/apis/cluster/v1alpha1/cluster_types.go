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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const ClusterFinalizer = "cluster.cluster.cnct.sds.samsung.com"

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	// Cluster network configuration
	ClusterNetwork ClusterNetworkingConfig `json:"clusterNetwork"`

	KubernetesVersion string `json:"kubernetesVersion"`
}

// ClusterNetworkingConfig specifies the different networking
// parameters for a cluster.
type ClusterNetworkingConfig struct {
	// The network ranges from which service VIPs are allocated.
	Services NetworkRanges `json:"services"`

	// The network ranges from which POD networks are allocated.
	Pods NetworkRanges `json:"pods"`

	// Domain name for services.
	ServiceDomain string `json:"serviceDomain"`
}

// NetworkRanges represents ranges of network addresses.
type NetworkRanges struct {
	CIDRBlock string `json:"cidrBlock"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	// When was this status last observed
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`
	// API endpoint
	APIEndpoint string `json:"apiendpoint,omitempty"`
	// Cluster status
	Phase common.ClusterStatusPhase `json:"phase,omitempty"`
}

// APIEndpoint represents a reachable Kubernetes API endpoint.
type APIEndpoint struct {
	// The hostname on which the API server is serving.
	Host string `json:"host"`

	// The port on which the API server is serving.
	Port int `json:"port"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster is the Schema for the clusters API
// +k8s:openapi-gen=true
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
