# Protocol Documentation
<a name="top"/>

## Table of Contents

- [api.proto](#api.proto)
    - [AdjustClusterMsg](#cmassh.AdjustClusterMsg)
    - [AdjustClusterMsg.SshRemoveMachineSpec](#cmassh.AdjustClusterMsg.SshRemoveMachineSpec)
    - [AdjustClusterReply](#cmassh.AdjustClusterReply)
    - [ClusterDetailItem](#cmassh.ClusterDetailItem)
    - [ClusterItem](#cmassh.ClusterItem)
    - [CreateClusterMsg](#cmassh.CreateClusterMsg)
    - [CreateClusterReply](#cmassh.CreateClusterReply)
    - [DeleteClusterMsg](#cmassh.DeleteClusterMsg)
    - [DeleteClusterReply](#cmassh.DeleteClusterReply)
    - [GetClusterListMsg](#cmassh.GetClusterListMsg)
    - [GetClusterListReply](#cmassh.GetClusterListReply)
    - [GetClusterMsg](#cmassh.GetClusterMsg)
    - [GetClusterReply](#cmassh.GetClusterReply)
    - [GetUpgradeClusterInformationMsg](#cmassh.GetUpgradeClusterInformationMsg)
    - [GetUpgradeClusterInformationReply](#cmassh.GetUpgradeClusterInformationReply)
    - [GetVersionMsg](#cmassh.GetVersionMsg)
    - [GetVersionReply](#cmassh.GetVersionReply)
    - [GetVersionReply.VersionInformation](#cmassh.GetVersionReply.VersionInformation)
    - [KubernetesLabel](#cmassh.KubernetesLabel)
    - [SshMachineSpec](#cmassh.SshMachineSpec)
    - [UpgradeClusterMsg](#cmassh.UpgradeClusterMsg)
    - [UpgradeClusterReply](#cmassh.UpgradeClusterReply)
  
    - [ClusterStatus](#cmassh.ClusterStatus)
  
  
    - [Cluster](#cmassh.Cluster)
  

- [api.proto](#api.proto)
    - [AdjustClusterMsg](#cmassh.AdjustClusterMsg)
    - [AdjustClusterMsg.SshRemoveMachineSpec](#cmassh.AdjustClusterMsg.SshRemoveMachineSpec)
    - [AdjustClusterReply](#cmassh.AdjustClusterReply)
    - [ClusterDetailItem](#cmassh.ClusterDetailItem)
    - [ClusterItem](#cmassh.ClusterItem)
    - [CreateClusterMsg](#cmassh.CreateClusterMsg)
    - [CreateClusterReply](#cmassh.CreateClusterReply)
    - [DeleteClusterMsg](#cmassh.DeleteClusterMsg)
    - [DeleteClusterReply](#cmassh.DeleteClusterReply)
    - [GetClusterListMsg](#cmassh.GetClusterListMsg)
    - [GetClusterListReply](#cmassh.GetClusterListReply)
    - [GetClusterMsg](#cmassh.GetClusterMsg)
    - [GetClusterReply](#cmassh.GetClusterReply)
    - [GetUpgradeClusterInformationMsg](#cmassh.GetUpgradeClusterInformationMsg)
    - [GetUpgradeClusterInformationReply](#cmassh.GetUpgradeClusterInformationReply)
    - [GetVersionMsg](#cmassh.GetVersionMsg)
    - [GetVersionReply](#cmassh.GetVersionReply)
    - [GetVersionReply.VersionInformation](#cmassh.GetVersionReply.VersionInformation)
    - [KubernetesLabel](#cmassh.KubernetesLabel)
    - [SshMachineSpec](#cmassh.SshMachineSpec)
    - [UpgradeClusterMsg](#cmassh.UpgradeClusterMsg)
    - [UpgradeClusterReply](#cmassh.UpgradeClusterReply)
  
    - [ClusterStatus](#cmassh.ClusterStatus)
  
  
    - [Cluster](#cmassh.Cluster)
  

- [Scalar Value Types](#scalar-value-types)



<a name="api.proto"/>
<p align="right"><a href="#top">Top</a></p>

## api.proto



<a name="cmassh.AdjustClusterMsg"/>

### AdjustClusterMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | What is the cluster that we are considering for upgrade |
| add_nodes | [SshMachineSpec](#cmassh.SshMachineSpec) | repeated | Machines which we want to add to the cluster |
| remove_nodes | [AdjustClusterMsg.SshRemoveMachineSpec](#cmassh.AdjustClusterMsg.SshRemoveMachineSpec) | repeated | Machines which we want to remove from the cluster |






<a name="cmassh.AdjustClusterMsg.SshRemoveMachineSpec"/>

### AdjustClusterMsg.SshRemoveMachineSpec



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| host | [string](#string) |  | The host for SSH access |






<a name="cmassh.AdjustClusterReply"/>

### AdjustClusterReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Was this a successful request |






<a name="cmassh.ClusterDetailItem"/>

### ClusterDetailItem



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | ID of the cluster |
| name | [string](#string) |  | Name of the cluster |
| status_message | [string](#string) |  | Additional information about the status of the cluster |
| kubeconfig | [string](#string) |  | What is the kubeconfig to connect to the cluster |
| status | [ClusterStatus](#cmassh.ClusterStatus) |  | The status of the cluster |






<a name="cmassh.ClusterItem"/>

### ClusterItem



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | ID of the cluster |
| name | [string](#string) |  | Name of the cluster |
| status_message | [string](#string) |  | Additional information about the status of the cluster |
| status | [ClusterStatus](#cmassh.ClusterStatus) |  | The status of the cluster |






<a name="cmassh.CreateClusterMsg"/>

### CreateClusterMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of the cluster to be provisioned |
| k8s_version | [string](#string) |  | The version of Kubernetes for worker nodes. Control plane versions are determined by the MachineSpec. |
| high_availability | [bool](#bool) |  | Whether or not the cluster is HA |
| network_fabric | [string](#string) |  | The fabric to be used |
| control_plane_nodes | [SshMachineSpec](#cmassh.SshMachineSpec) | repeated | Machines which comprise the cluster |
| worker_nodes | [SshMachineSpec](#cmassh.SshMachineSpec) | repeated | Machines which comprise the cluster |
| api_endpoint | [string](#string) |  | This should be a value like ip:port that will be a virtual IP/port Passed back to external customers to be able to communicate to the cluster |
| private_key | [string](#string) |  | Private key used to ssh into machines |






<a name="cmassh.CreateClusterReply"/>

### CreateClusterReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Whether or not the cluster was provisioned by this request |
| cluster | [ClusterItem](#cmassh.ClusterItem) |  | The details of the cluster request response |






<a name="cmassh.DeleteClusterMsg"/>

### DeleteClusterMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | What is the cluster&#39;s name to destroy |






<a name="cmassh.DeleteClusterReply"/>

### DeleteClusterReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Could the cluster be destroyed |
| status | [string](#string) |  | Status of the request |






<a name="cmassh.GetClusterListMsg"/>

### GetClusterListMsg







<a name="cmassh.GetClusterListReply"/>

### GetClusterListReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Is the cluster in the system |
| clusters | [ClusterItem](#cmassh.ClusterItem) | repeated | List of clusters |






<a name="cmassh.GetClusterMsg"/>

### GetClusterMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of the cluster to be looked up |






<a name="cmassh.GetClusterReply"/>

### GetClusterReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Is the cluster in the system |
| cluster | [ClusterDetailItem](#cmassh.ClusterDetailItem) |  |  |






<a name="cmassh.GetUpgradeClusterInformationMsg"/>

### GetUpgradeClusterInformationMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | What is the cluster that we are considering for upgrade |






<a name="cmassh.GetUpgradeClusterInformationReply"/>

### GetUpgradeClusterInformationReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Can the cluster be upgraded |
| versions | [string](#string) | repeated | What versions are possible right now |






<a name="cmassh.GetVersionMsg"/>

### GetVersionMsg
Get version of API Server






<a name="cmassh.GetVersionReply"/>

### GetVersionReply
Reply for version request


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | If operation was OK |
| version_information | [GetVersionReply.VersionInformation](#cmassh.GetVersionReply.VersionInformation) |  | Version Information |






<a name="cmassh.GetVersionReply.VersionInformation"/>

### GetVersionReply.VersionInformation



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| git_version | [string](#string) |  | The tag on the git repository |
| git_commit | [string](#string) |  | The hash of the git commit |
| git_tree_state | [string](#string) |  | Whether or not the tree was clean when built |
| build_date | [string](#string) |  | Date of build |
| go_version | [string](#string) |  | Version of go used to compile |
| compiler | [string](#string) |  | Compiler used |
| platform | [string](#string) |  | Platform it was compiled for / running on |






<a name="cmassh.KubernetesLabel"/>

### KubernetesLabel



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of a label |
| value | [string](#string) |  | The value of a label |






<a name="cmassh.SshMachineSpec"/>

### SshMachineSpec
The specification for a specific node


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| username | [string](#string) |  | The username for SSH access |
| host | [string](#string) |  | The host for SSH access |
| port | [int32](#int32) |  | The port for SSH access |
| password | [string](#string) |  | The password for SSH access. This is not implemented within the clusterapi because without known_host support a MITM can get the password. A MITM is still a problem for key based authentication; even if they do not get the key they can still impersonate the machine. TODO: implement known_hosts. |
| labels | [KubernetesLabel](#cmassh.KubernetesLabel) | repeated | The labels for the machines |
| publichost | [string](#string) |  | optional public host ip |






<a name="cmassh.UpgradeClusterMsg"/>

### UpgradeClusterMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | What is the cluster that we are considering for upgrade |
| version | [string](#string) |  | What version are we upgrading to? |






<a name="cmassh.UpgradeClusterReply"/>

### UpgradeClusterReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Was this a successful request |





 


<a name="cmassh.ClusterStatus"/>

### ClusterStatus


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 | Not set |
| PROVISIONING | 1 | The PROVISIONING state indicates the cluster is being created. |
| RUNNING | 2 | The RUNNING state indicates the cluster has been created and is fully usable. |
| RECONCILING | 3 | The RECONCILING state indicates that some work is actively being done on the cluster, such as upgrading the master or node software. |
| STOPPING | 4 | The STOPPING state indicates the cluster is being deleted |
| ERROR | 5 | The ERROR state indicates the cluster may be unusable |
| DEGRADED | 6 | The DEGRADED state indicates the cluster requires user action to restore full functionality |


 

 


<a name="cmassh.Cluster"/>

### Cluster


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateCluster | [CreateClusterMsg](#cmassh.CreateClusterMsg) | [CreateClusterReply](#cmassh.CreateClusterMsg) | Will provision a cluster |
| GetCluster | [GetClusterMsg](#cmassh.GetClusterMsg) | [GetClusterReply](#cmassh.GetClusterMsg) | Will retrieve the status of a cluster and its kubeconfig for connectivity |
| DeleteCluster | [DeleteClusterMsg](#cmassh.DeleteClusterMsg) | [DeleteClusterReply](#cmassh.DeleteClusterMsg) | Will delete a cluster |
| GetClusterList | [GetClusterListMsg](#cmassh.GetClusterListMsg) | [GetClusterListReply](#cmassh.GetClusterListMsg) | Will retrieve a list of clusters |
| GetVersionInformation | [GetVersionMsg](#cmassh.GetVersionMsg) | [GetVersionReply](#cmassh.GetVersionMsg) | Will return version information about api server |
| AdjustClusterNodes | [AdjustClusterMsg](#cmassh.AdjustClusterMsg) | [AdjustClusterReply](#cmassh.AdjustClusterMsg) | Will adjust a provision a cluster |
| GetUpgradeClusterInformation | [GetUpgradeClusterInformationMsg](#cmassh.GetUpgradeClusterInformationMsg) | [GetUpgradeClusterInformationReply](#cmassh.GetUpgradeClusterInformationMsg) | Will return upgrade options for a given cluster |
| UpgradeCluster | [UpgradeClusterMsg](#cmassh.UpgradeClusterMsg) | [UpgradeClusterReply](#cmassh.UpgradeClusterMsg) | Will attempt to upgrade a cluster |

 



<a name="api.proto"/>
<p align="right"><a href="#top">Top</a></p>

## api.proto



<a name="cmassh.AdjustClusterMsg"/>

### AdjustClusterMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | What is the cluster that we are considering for upgrade |
| add_nodes | [SshMachineSpec](#cmassh.SshMachineSpec) | repeated | Machines which we want to add to the cluster |
| remove_nodes | [AdjustClusterMsg.SshRemoveMachineSpec](#cmassh.AdjustClusterMsg.SshRemoveMachineSpec) | repeated | Machines which we want to remove from the cluster |






<a name="cmassh.AdjustClusterMsg.SshRemoveMachineSpec"/>

### AdjustClusterMsg.SshRemoveMachineSpec



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| host | [string](#string) |  | The host for SSH access |






<a name="cmassh.AdjustClusterReply"/>

### AdjustClusterReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Was this a successful request |






<a name="cmassh.ClusterDetailItem"/>

### ClusterDetailItem



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | ID of the cluster |
| name | [string](#string) |  | Name of the cluster |
| status_message | [string](#string) |  | Additional information about the status of the cluster |
| kubeconfig | [string](#string) |  | What is the kubeconfig to connect to the cluster |
| status | [ClusterStatus](#cmassh.ClusterStatus) |  | The status of the cluster |






<a name="cmassh.ClusterItem"/>

### ClusterItem



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | ID of the cluster |
| name | [string](#string) |  | Name of the cluster |
| status_message | [string](#string) |  | Additional information about the status of the cluster |
| status | [ClusterStatus](#cmassh.ClusterStatus) |  | The status of the cluster |






<a name="cmassh.CreateClusterMsg"/>

### CreateClusterMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of the cluster to be provisioned |
| k8s_version | [string](#string) |  | The version of Kubernetes for worker nodes. Control plane versions are determined by the MachineSpec. |
| high_availability | [bool](#bool) |  | Whether or not the cluster is HA |
| network_fabric | [string](#string) |  | The fabric to be used |
| control_plane_nodes | [SshMachineSpec](#cmassh.SshMachineSpec) | repeated | Machines which comprise the cluster |
| worker_nodes | [SshMachineSpec](#cmassh.SshMachineSpec) | repeated | Machines which comprise the cluster |
| api_endpoint | [string](#string) |  | This should be a value like ip:port that will be a virtual IP/port Passed back to external customers to be able to communicate to the cluster |
| private_key | [string](#string) |  | Private key used to ssh into machines |






<a name="cmassh.CreateClusterReply"/>

### CreateClusterReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Whether or not the cluster was provisioned by this request |
| cluster | [ClusterItem](#cmassh.ClusterItem) |  | The details of the cluster request response |






<a name="cmassh.DeleteClusterMsg"/>

### DeleteClusterMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | What is the cluster&#39;s name to destroy |






<a name="cmassh.DeleteClusterReply"/>

### DeleteClusterReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Could the cluster be destroyed |
| status | [string](#string) |  | Status of the request |






<a name="cmassh.GetClusterListMsg"/>

### GetClusterListMsg







<a name="cmassh.GetClusterListReply"/>

### GetClusterListReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Is the cluster in the system |
| clusters | [ClusterItem](#cmassh.ClusterItem) | repeated | List of clusters |






<a name="cmassh.GetClusterMsg"/>

### GetClusterMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of the cluster to be looked up |






<a name="cmassh.GetClusterReply"/>

### GetClusterReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Is the cluster in the system |
| cluster | [ClusterDetailItem](#cmassh.ClusterDetailItem) |  |  |






<a name="cmassh.GetUpgradeClusterInformationMsg"/>

### GetUpgradeClusterInformationMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | What is the cluster that we are considering for upgrade |






<a name="cmassh.GetUpgradeClusterInformationReply"/>

### GetUpgradeClusterInformationReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Can the cluster be upgraded |
| versions | [string](#string) | repeated | What versions are possible right now |






<a name="cmassh.GetVersionMsg"/>

### GetVersionMsg
Get version of API Server






<a name="cmassh.GetVersionReply"/>

### GetVersionReply
Reply for version request


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | If operation was OK |
| version_information | [GetVersionReply.VersionInformation](#cmassh.GetVersionReply.VersionInformation) |  | Version Information |






<a name="cmassh.GetVersionReply.VersionInformation"/>

### GetVersionReply.VersionInformation



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| git_version | [string](#string) |  | The tag on the git repository |
| git_commit | [string](#string) |  | The hash of the git commit |
| git_tree_state | [string](#string) |  | Whether or not the tree was clean when built |
| build_date | [string](#string) |  | Date of build |
| go_version | [string](#string) |  | Version of go used to compile |
| compiler | [string](#string) |  | Compiler used |
| platform | [string](#string) |  | Platform it was compiled for / running on |






<a name="cmassh.KubernetesLabel"/>

### KubernetesLabel



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of a label |
| value | [string](#string) |  | The value of a label |






<a name="cmassh.SshMachineSpec"/>

### SshMachineSpec
The specification for a specific node


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| username | [string](#string) |  | The username for SSH access |
| host | [string](#string) |  | The host for SSH access |
| port | [int32](#int32) |  | The port for SSH access |
| password | [string](#string) |  | The password for SSH access. This is not implemented within the clusterapi because without known_host support a MITM can get the password. A MITM is still a problem for key based authentication; even if they do not get the key they can still impersonate the machine. TODO: implement known_hosts. |
| labels | [KubernetesLabel](#cmassh.KubernetesLabel) | repeated | The labels for the machines |
| publichost | [string](#string) |  | optional public host ip |






<a name="cmassh.UpgradeClusterMsg"/>

### UpgradeClusterMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | What is the cluster that we are considering for upgrade |
| version | [string](#string) |  | What version are we upgrading to? |






<a name="cmassh.UpgradeClusterReply"/>

### UpgradeClusterReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Was this a successful request |





 


<a name="cmassh.ClusterStatus"/>

### ClusterStatus


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 | Not set |
| PROVISIONING | 1 | The PROVISIONING state indicates the cluster is being created. |
| RUNNING | 2 | The RUNNING state indicates the cluster has been created and is fully usable. |
| RECONCILING | 3 | The RECONCILING state indicates that some work is actively being done on the cluster, such as upgrading the master or node software. |
| STOPPING | 4 | The STOPPING state indicates the cluster is being deleted |
| ERROR | 5 | The ERROR state indicates the cluster may be unusable |
| DEGRADED | 6 | The DEGRADED state indicates the cluster requires user action to restore full functionality |


 

 


<a name="cmassh.Cluster"/>

### Cluster


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateCluster | [CreateClusterMsg](#cmassh.CreateClusterMsg) | [CreateClusterReply](#cmassh.CreateClusterMsg) | Will provision a cluster |
| GetCluster | [GetClusterMsg](#cmassh.GetClusterMsg) | [GetClusterReply](#cmassh.GetClusterMsg) | Will retrieve the status of a cluster and its kubeconfig for connectivity |
| DeleteCluster | [DeleteClusterMsg](#cmassh.DeleteClusterMsg) | [DeleteClusterReply](#cmassh.DeleteClusterMsg) | Will delete a cluster |
| GetClusterList | [GetClusterListMsg](#cmassh.GetClusterListMsg) | [GetClusterListReply](#cmassh.GetClusterListMsg) | Will retrieve a list of clusters |
| GetVersionInformation | [GetVersionMsg](#cmassh.GetVersionMsg) | [GetVersionReply](#cmassh.GetVersionMsg) | Will return version information about api server |
| AdjustClusterNodes | [AdjustClusterMsg](#cmassh.AdjustClusterMsg) | [AdjustClusterReply](#cmassh.AdjustClusterMsg) | Will adjust a provision a cluster |
| GetUpgradeClusterInformation | [GetUpgradeClusterInformationMsg](#cmassh.GetUpgradeClusterInformationMsg) | [GetUpgradeClusterInformationReply](#cmassh.GetUpgradeClusterInformationMsg) | Will return upgrade options for a given cluster |
| UpgradeCluster | [UpgradeClusterMsg](#cmassh.UpgradeClusterMsg) | [UpgradeClusterReply](#cmassh.UpgradeClusterMsg) | Will attempt to upgrade a cluster |

 



## Scalar Value Types

| .proto Type | Notes | C++ Type | Java Type | Python Type |
| ----------- | ----- | -------- | --------- | ----------- |
| <a name="double" /> double |  | double | double | float |
| <a name="float" /> float |  | float | float | float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long |
| <a name="bool" /> bool |  | bool | boolean | boolean |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str |

