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

package common

type MachineStatusPhase string

const (

	// resource is creating
	ProvisioningMachinePhase MachineStatusPhase = "CreatingMachine"

	// resource is ugrading
	UpgradingMachinePhase MachineStatusPhase = "UpgradingMachine"

	// resource is deleting
	DeletingMachinePhase MachineStatusPhase = "DeletingMachine"

	// resources are ready
	ReadyMachinePhase MachineStatusPhase = "ReadyMachine"

	// resources are in error state
	ErrorMachinePhase MachineStatusPhase = "ErrorMachine"
)

type ClusterStatusPhase string

const (
	UnspecifiedClusterPhase ClusterStatusPhase = "Unspecified"

	// The RUNNING state indicates the cluster has been created and is fully usable.
	RunningClusterPhase ClusterStatusPhase = "RunningCluster"

	// The RECONCILING state indicates that some work is actively being done on the cluster, such as
	// upgrading the master or node software.
	ReconcilingClusterPhase ClusterStatusPhase = "ReconcilingCluster"

	// The STOPPING state indicates the cluster is being deleted
	StoppingClusterPhase ClusterStatusPhase = "StoppingCluster"

	// The ERROR state indicates the cluster may be unusable
	ErrorClusterPhase ClusterStatusPhase = "ErrorCluster"
)

type ClusterStatusError string

const (
	// InvalidConfigurationClusterError indicates that the cluster
	// configuration is invalid.
	InvalidConfigurationClusterError ClusterStatusError = "InvalidConfiguration"

	// UnsupportedChangeClusterError indicates that the cluster
	// spec has been updated in an unsupported way. That cannot be
	// reconciled.
	UnsupportedChangeClusterError ClusterStatusError = "UnsupportedChange"

	// CreateClusterError indicates that an error was encountered
	// when trying to create the cluster.
	CreateClusterError ClusterStatusError = "CreateError"

	// UpdateClusterError indicates that an error was encountered
	// when trying to update the cluster.
	UpdateClusterError ClusterStatusError = "UpdateError"

	// DeleteClusterError indicates that an error was encountered
	// when trying to delete the cluster.
	DeleteClusterError ClusterStatusError = "DeleteError"
)

type MachineRoles string

const (
	MachineRoleMaster MachineRoles = "master"
	MachineRoleEtcd   MachineRoles = "etcd"
	MachineRoleWorker MachineRoles = "worker"
)

type ControllerEvents string

const (
	// ErrResourceFailed is used as part of the Event 'reason' when a resource fails
	ErrResourceFailed ControllerEvents = "ErrResourceFailed"
	// ResourceStateChange is used as part of Event 'reason' when a resource changes phase (non error)
	ResourceStateChange ControllerEvents = "ResourceStateChange"

	// MessageResourceExists is the message used for Events when a resource fails to sync
	MessageResourceFailed ControllerEvents = "Resource %q failed to reconcile"
	// MessageResourceStateChange is the message used for an Event fired when a resource
	// changes phase (non error)
	MessageResourceStateChange ControllerEvents = "Resource %q changed state to %q"
)

const (
	ApiEnpointPort = "6443"
)
