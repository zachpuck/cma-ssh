# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [api.proto](#api.proto)
    - [AddNodePoolMsg](#cmassh.AddNodePoolMsg)
    - [AddNodePoolReply](#cmassh.AddNodePoolReply)
    - [ClusterDetailItem](#cmassh.ClusterDetailItem)
    - [ClusterItem](#cmassh.ClusterItem)
    - [ControlPlaneMachineSpec](#cmassh.ControlPlaneMachineSpec)
    - [CreateClusterMsg](#cmassh.CreateClusterMsg)
    - [CreateClusterReply](#cmassh.CreateClusterReply)
    - [DeleteClusterMsg](#cmassh.DeleteClusterMsg)
    - [DeleteClusterReply](#cmassh.DeleteClusterReply)
    - [DeleteNodePoolMsg](#cmassh.DeleteNodePoolMsg)
    - [DeleteNodePoolReply](#cmassh.DeleteNodePoolReply)
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
    - [MachineSpec](#cmassh.MachineSpec)
    - [ScaleNodePoolMsg](#cmassh.ScaleNodePoolMsg)
    - [ScaleNodePoolReply](#cmassh.ScaleNodePoolReply)
    - [ScaleNodePoolSpec](#cmassh.ScaleNodePoolSpec)
    - [UpgradeClusterMsg](#cmassh.UpgradeClusterMsg)
    - [UpgradeClusterReply](#cmassh.UpgradeClusterReply)
  
    - [ClusterStatus](#cmassh.ClusterStatus)
  
  
    - [Cluster](#cmassh.Cluster)
  

- [api.proto](#api.proto)
    - [AddNodePoolMsg](#cmassh.AddNodePoolMsg)
    - [AddNodePoolReply](#cmassh.AddNodePoolReply)
    - [ClusterDetailItem](#cmassh.ClusterDetailItem)
    - [ClusterItem](#cmassh.ClusterItem)
    - [ControlPlaneMachineSpec](#cmassh.ControlPlaneMachineSpec)
    - [CreateClusterMsg](#cmassh.CreateClusterMsg)
    - [CreateClusterReply](#cmassh.CreateClusterReply)
    - [DeleteClusterMsg](#cmassh.DeleteClusterMsg)
    - [DeleteClusterReply](#cmassh.DeleteClusterReply)
    - [DeleteNodePoolMsg](#cmassh.DeleteNodePoolMsg)
    - [DeleteNodePoolReply](#cmassh.DeleteNodePoolReply)
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
    - [MachineSpec](#cmassh.MachineSpec)
    - [ScaleNodePoolMsg](#cmassh.ScaleNodePoolMsg)
    - [ScaleNodePoolReply](#cmassh.ScaleNodePoolReply)
    - [ScaleNodePoolSpec](#cmassh.ScaleNodePoolSpec)
    - [UpgradeClusterMsg](#cmassh.UpgradeClusterMsg)
    - [UpgradeClusterReply](#cmassh.UpgradeClusterReply)
  
    - [ClusterStatus](#cmassh.ClusterStatus)
  
  
    - [Cluster](#cmassh.Cluster)
  

- [Scalar Value Types](#scalar-value-types)



<a name="api.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api.proto



<a name="cmassh.AddNodePoolMsg"></a>

### AddNodePoolMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| clusterName | [string](#string) |  | What is the cluster to add node pools to |
| worker_node_pools | [MachineSpec](#cmassh.MachineSpec) | repeated | What Machines to add to the cluster |






<a name="cmassh.AddNodePoolReply"></a>

### AddNodePoolReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Whether or not the node pool was provisioned by this request |






<a name="cmassh.ClusterDetailItem"></a>

### ClusterDetailItem



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of the cluster |
| status_message | [string](#string) |  | Additional information about the status of the cluster |
| kubeconfig | [string](#string) |  | What is the kubeconfig to connect to the cluster |
| status | [ClusterStatus](#cmassh.ClusterStatus) |  | The status of the cluster |






<a name="cmassh.ClusterItem"></a>

### ClusterItem



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of the cluster |
| status_message | [string](#string) |  | Additional information about the status of the cluster |
| status | [ClusterStatus](#cmassh.ClusterStatus) |  | The status of the cluster |






<a name="cmassh.ControlPlaneMachineSpec"></a>

### ControlPlaneMachineSpec
The specification for a set of control plane machines


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| labels | [KubernetesLabel](#cmassh.KubernetesLabel) | repeated | The labels for the control plane machines |
| instanceType | [string](#string) |  | Type of machines to provision (standard or gpu) |
| count | [int32](#int32) |  | The number of machines |






<a name="cmassh.CreateClusterMsg"></a>

### CreateClusterMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of the cluster to be provisioned |
| k8s_version | [string](#string) |  | The version of Kubernetes for worker nodes. Control plane versions are determined by the MachineSpec. |
| control_plane_nodes | [ControlPlaneMachineSpec](#cmassh.ControlPlaneMachineSpec) |  | Machines which comprise the cluster control plane |
| worker_node_pools | [MachineSpec](#cmassh.MachineSpec) | repeated | Machines which comprise the cluster |






<a name="cmassh.CreateClusterReply"></a>

### CreateClusterReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Whether or not the cluster was provisioned by this request |
| cluster | [ClusterItem](#cmassh.ClusterItem) |  | The details of the cluster request response |






<a name="cmassh.DeleteClusterMsg"></a>

### DeleteClusterMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | What is the cluster&#39;s name to destroy |






<a name="cmassh.DeleteClusterReply"></a>

### DeleteClusterReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Could the cluster be destroyed |
| status | [string](#string) |  | Status of the request |






<a name="cmassh.DeleteNodePoolMsg"></a>

### DeleteNodePoolMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| clusterName | [string](#string) |  | What is the cluster to delete node pools |
| node_pool_names | [string](#string) | repeated | What is the node pool names to delete |






<a name="cmassh.DeleteNodePoolReply"></a>

### DeleteNodePoolReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Was this a successful request |






<a name="cmassh.GetClusterListMsg"></a>

### GetClusterListMsg







<a name="cmassh.GetClusterListReply"></a>

### GetClusterListReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Is the cluster in the system |
| clusters | [ClusterItem](#cmassh.ClusterItem) | repeated | List of clusters |






<a name="cmassh.GetClusterMsg"></a>

### GetClusterMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of the cluster to be looked up |






<a name="cmassh.GetClusterReply"></a>

### GetClusterReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Is the cluster in the system |
| cluster | [ClusterDetailItem](#cmassh.ClusterDetailItem) |  |  |






<a name="cmassh.GetUpgradeClusterInformationMsg"></a>

### GetUpgradeClusterInformationMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | What is the cluster that we are considering for upgrade |






<a name="cmassh.GetUpgradeClusterInformationReply"></a>

### GetUpgradeClusterInformationReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Can the cluster be upgraded |
| versions | [string](#string) | repeated | What versions are possible right now |






<a name="cmassh.GetVersionMsg"></a>

### GetVersionMsg
Get version of API Server






<a name="cmassh.GetVersionReply"></a>

### GetVersionReply
Reply for version request


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | If operation was OK |
| version_information | [GetVersionReply.VersionInformation](#cmassh.GetVersionReply.VersionInformation) |  | Version Information |






<a name="cmassh.GetVersionReply.VersionInformation"></a>

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






<a name="cmassh.KubernetesLabel"></a>

### KubernetesLabel



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of a label |
| value | [string](#string) |  | The value of a label |






<a name="cmassh.MachineSpec"></a>

### MachineSpec
The specification for a set of machines


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the machine set |
| labels | [KubernetesLabel](#cmassh.KubernetesLabel) | repeated | The labels for the machine set |
| instanceType | [string](#string) |  | Type of machines to provision (standard or gpu) |
| count | [int32](#int32) |  | The number of machines |






<a name="cmassh.ScaleNodePoolMsg"></a>

### ScaleNodePoolMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| clusterName | [string](#string) |  | What is the name of the cluster to scale a node pool |
| node_pools | [ScaleNodePoolSpec](#cmassh.ScaleNodePoolSpec) | repeated | What node pools to scale |






<a name="cmassh.ScaleNodePoolReply"></a>

### ScaleNodePoolReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Was this a successful request |






<a name="cmassh.ScaleNodePoolSpec"></a>

### ScaleNodePoolSpec



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | What is the node pool name to scale |
| count | [int32](#int32) |  | Number of machines to scale |






<a name="cmassh.UpgradeClusterMsg"></a>

### UpgradeClusterMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | What is the cluster that we are considering for upgrade |
| version | [string](#string) |  | What version are we upgrading to? |






<a name="cmassh.UpgradeClusterReply"></a>

### UpgradeClusterReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Was this a successful request |





 


<a name="cmassh.ClusterStatus"></a>

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


 

 


<a name="cmassh.Cluster"></a>

### Cluster


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateCluster | [CreateClusterMsg](#cmassh.CreateClusterMsg) | [CreateClusterReply](#cmassh.CreateClusterReply) | Will provision a cluster |
| GetCluster | [GetClusterMsg](#cmassh.GetClusterMsg) | [GetClusterReply](#cmassh.GetClusterReply) | Will retrieve the status of a cluster and its kubeconfig for connectivity |
| DeleteCluster | [DeleteClusterMsg](#cmassh.DeleteClusterMsg) | [DeleteClusterReply](#cmassh.DeleteClusterReply) | Will delete a cluster |
| GetClusterList | [GetClusterListMsg](#cmassh.GetClusterListMsg) | [GetClusterListReply](#cmassh.GetClusterListReply) | Will retrieve a list of clusters |
| GetVersionInformation | [GetVersionMsg](#cmassh.GetVersionMsg) | [GetVersionReply](#cmassh.GetVersionReply) | Will return version information about api server |
| AddNodePool | [AddNodePoolMsg](#cmassh.AddNodePoolMsg) | [AddNodePoolReply](#cmassh.AddNodePoolReply) | Will add node pool to a provisioned cluster |
| DeleteNodePool | [DeleteNodePoolMsg](#cmassh.DeleteNodePoolMsg) | [DeleteNodePoolReply](#cmassh.DeleteNodePoolReply) | Will delete a node pool from a provisioned cluster |
| ScaleNodePool | [ScaleNodePoolMsg](#cmassh.ScaleNodePoolMsg) | [ScaleNodePoolReply](#cmassh.ScaleNodePoolReply) | Will scale the number of machines in a node pool for a provisioned cluster |
| GetUpgradeClusterInformation | [GetUpgradeClusterInformationMsg](#cmassh.GetUpgradeClusterInformationMsg) | [GetUpgradeClusterInformationReply](#cmassh.GetUpgradeClusterInformationReply) | Will return upgrade options for a given cluster |
| UpgradeCluster | [UpgradeClusterMsg](#cmassh.UpgradeClusterMsg) | [UpgradeClusterReply](#cmassh.UpgradeClusterReply) | Will attempt to upgrade a cluster |

 



<a name="api.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api.proto



<a name="cmassh.AddNodePoolMsg"></a>

### AddNodePoolMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| clusterName | [string](#string) |  | What is the cluster to add node pools to |
| worker_node_pools | [MachineSpec](#cmassh.MachineSpec) | repeated | What Machines to add to the cluster |






<a name="cmassh.AddNodePoolReply"></a>

### AddNodePoolReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Whether or not the node pool was provisioned by this request |






<a name="cmassh.ClusterDetailItem"></a>

### ClusterDetailItem



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of the cluster |
| status_message | [string](#string) |  | Additional information about the status of the cluster |
| kubeconfig | [string](#string) |  | What is the kubeconfig to connect to the cluster |
| status | [ClusterStatus](#cmassh.ClusterStatus) |  | The status of the cluster |






<a name="cmassh.ClusterItem"></a>

### ClusterItem



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of the cluster |
| status_message | [string](#string) |  | Additional information about the status of the cluster |
| status | [ClusterStatus](#cmassh.ClusterStatus) |  | The status of the cluster |






<a name="cmassh.ControlPlaneMachineSpec"></a>

### ControlPlaneMachineSpec
The specification for a set of control plane machines


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| labels | [KubernetesLabel](#cmassh.KubernetesLabel) | repeated | The labels for the control plane machines |
| instanceType | [string](#string) |  | Type of machines to provision (standard or gpu) |
| count | [int32](#int32) |  | The number of machines |






<a name="cmassh.CreateClusterMsg"></a>

### CreateClusterMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of the cluster to be provisioned |
| k8s_version | [string](#string) |  | The version of Kubernetes for worker nodes. Control plane versions are determined by the MachineSpec. |
| control_plane_nodes | [ControlPlaneMachineSpec](#cmassh.ControlPlaneMachineSpec) |  | Machines which comprise the cluster control plane |
| worker_node_pools | [MachineSpec](#cmassh.MachineSpec) | repeated | Machines which comprise the cluster |






<a name="cmassh.CreateClusterReply"></a>

### CreateClusterReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Whether or not the cluster was provisioned by this request |
| cluster | [ClusterItem](#cmassh.ClusterItem) |  | The details of the cluster request response |






<a name="cmassh.DeleteClusterMsg"></a>

### DeleteClusterMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | What is the cluster&#39;s name to destroy |






<a name="cmassh.DeleteClusterReply"></a>

### DeleteClusterReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Could the cluster be destroyed |
| status | [string](#string) |  | Status of the request |






<a name="cmassh.DeleteNodePoolMsg"></a>

### DeleteNodePoolMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| clusterName | [string](#string) |  | What is the cluster to delete node pools |
| node_pool_names | [string](#string) | repeated | What is the node pool names to delete |






<a name="cmassh.DeleteNodePoolReply"></a>

### DeleteNodePoolReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Was this a successful request |






<a name="cmassh.GetClusterListMsg"></a>

### GetClusterListMsg







<a name="cmassh.GetClusterListReply"></a>

### GetClusterListReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Is the cluster in the system |
| clusters | [ClusterItem](#cmassh.ClusterItem) | repeated | List of clusters |






<a name="cmassh.GetClusterMsg"></a>

### GetClusterMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of the cluster to be looked up |






<a name="cmassh.GetClusterReply"></a>

### GetClusterReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Is the cluster in the system |
| cluster | [ClusterDetailItem](#cmassh.ClusterDetailItem) |  |  |






<a name="cmassh.GetUpgradeClusterInformationMsg"></a>

### GetUpgradeClusterInformationMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | What is the cluster that we are considering for upgrade |






<a name="cmassh.GetUpgradeClusterInformationReply"></a>

### GetUpgradeClusterInformationReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Can the cluster be upgraded |
| versions | [string](#string) | repeated | What versions are possible right now |






<a name="cmassh.GetVersionMsg"></a>

### GetVersionMsg
Get version of API Server






<a name="cmassh.GetVersionReply"></a>

### GetVersionReply
Reply for version request


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | If operation was OK |
| version_information | [GetVersionReply.VersionInformation](#cmassh.GetVersionReply.VersionInformation) |  | Version Information |






<a name="cmassh.GetVersionReply.VersionInformation"></a>

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






<a name="cmassh.KubernetesLabel"></a>

### KubernetesLabel



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of a label |
| value | [string](#string) |  | The value of a label |






<a name="cmassh.MachineSpec"></a>

### MachineSpec
The specification for a set of machines


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the machine set |
| labels | [KubernetesLabel](#cmassh.KubernetesLabel) | repeated | The labels for the machine set |
| instanceType | [string](#string) |  | Type of machines to provision (standard or gpu) |
| count | [int32](#int32) |  | The number of machines |






<a name="cmassh.ScaleNodePoolMsg"></a>

### ScaleNodePoolMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| clusterName | [string](#string) |  | What is the name of the cluster to scale a node pool |
| node_pools | [ScaleNodePoolSpec](#cmassh.ScaleNodePoolSpec) | repeated | What node pools to scale |






<a name="cmassh.ScaleNodePoolReply"></a>

### ScaleNodePoolReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Was this a successful request |






<a name="cmassh.ScaleNodePoolSpec"></a>

### ScaleNodePoolSpec



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | What is the node pool name to scale |
| count | [int32](#int32) |  | Number of machines to scale |






<a name="cmassh.UpgradeClusterMsg"></a>

### UpgradeClusterMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | What is the cluster that we are considering for upgrade |
| version | [string](#string) |  | What version are we upgrading to? |






<a name="cmassh.UpgradeClusterReply"></a>

### UpgradeClusterReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  | Was this a successful request |





 


<a name="cmassh.ClusterStatus"></a>

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


 

 


<a name="cmassh.Cluster"></a>

### Cluster


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateCluster | [CreateClusterMsg](#cmassh.CreateClusterMsg) | [CreateClusterReply](#cmassh.CreateClusterReply) | Will provision a cluster |
| GetCluster | [GetClusterMsg](#cmassh.GetClusterMsg) | [GetClusterReply](#cmassh.GetClusterReply) | Will retrieve the status of a cluster and its kubeconfig for connectivity |
| DeleteCluster | [DeleteClusterMsg](#cmassh.DeleteClusterMsg) | [DeleteClusterReply](#cmassh.DeleteClusterReply) | Will delete a cluster |
| GetClusterList | [GetClusterListMsg](#cmassh.GetClusterListMsg) | [GetClusterListReply](#cmassh.GetClusterListReply) | Will retrieve a list of clusters |
| GetVersionInformation | [GetVersionMsg](#cmassh.GetVersionMsg) | [GetVersionReply](#cmassh.GetVersionReply) | Will return version information about api server |
| AddNodePool | [AddNodePoolMsg](#cmassh.AddNodePoolMsg) | [AddNodePoolReply](#cmassh.AddNodePoolReply) | Will add node pool to a provisioned cluster |
| DeleteNodePool | [DeleteNodePoolMsg](#cmassh.DeleteNodePoolMsg) | [DeleteNodePoolReply](#cmassh.DeleteNodePoolReply) | Will delete a node pool from a provisioned cluster |
| ScaleNodePool | [ScaleNodePoolMsg](#cmassh.ScaleNodePoolMsg) | [ScaleNodePoolReply](#cmassh.ScaleNodePoolReply) | Will scale the number of machines in a node pool for a provisioned cluster |
| GetUpgradeClusterInformation | [GetUpgradeClusterInformationMsg](#cmassh.GetUpgradeClusterInformationMsg) | [GetUpgradeClusterInformationReply](#cmassh.GetUpgradeClusterInformationReply) | Will return upgrade options for a given cluster |
| UpgradeCluster | [UpgradeClusterMsg](#cmassh.UpgradeClusterMsg) | [UpgradeClusterReply](#cmassh.UpgradeClusterReply) | Will attempt to upgrade a cluster |

 



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

