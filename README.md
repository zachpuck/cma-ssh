
# cma-ssh
[![Build Status](https://jenkins.migrations.cnct.io/buildStatus/icon?job=cma-ssh/master)](https://jenkins.migrations.cnct.io/job/cma-ssh/job/master/)

`cma-ssh` is an operator which manages the lifecycle of Kubernetes clusters
(i.e. `CnctCluster` resources) and machines (`CnctMachine`). 

## Developement

The Kubernetes cluster on which the `cma-ssh` is installed must
have network access to a MAAS server. Within the CNCT lab this
means you must be in the Seattle office or logged onto the VPN.
Additionally you will need  to 
[generate an API Key][generate-an-api-key] using the MAAS GUI.

To test `cma-ssh` you can use `kind` and `helm`. For example:

```bash
kind create cluster
export KUBECONFIG="$(kind get kubeconfig-path --name="1")"

kubectl create clusterrolebinding superpowers --clusterrole=cluster-admin --user=system:serviceaccount:kube-system:default
kubectl create rolebinding superpowers --clusterrole=cluster-admin --user=system:serviceaccount:kube-system:default

helm init

# Set the `maas.apiKey` value for your user.
vi deployments/helm/cma-ssh/values.yaml

helm install --name cma-ssh deployments/helm/cma-ssh/
kubectl get pods --watch
```

# Deprecated

The instructions below are deprecated as we move towards a cloud-init approach
to configuration instead of ssh.

## Overview

The cma-ssh repo provides a helper API for [cluster-manager-api](https://github.com/samsung-cnct/cluster-manager-api) by utilizing ssh to interact with virtual machines for kubernetes cluster create, upgrade, add node, and delete.

### Getting started

See [Protocol Documentation](https://github.com/samsung-cnct/cma-ssh/blob/master/docs/api-generated/api.md)
- [open api in swagger ui](http://petstore.swagger.io/?url=https://raw.githubusercontent.com/samsung-cnct/cma-ssh/master/assets/generated/swagger/api.swagger.json)
- [open api in swagger editor](https://editor.swagger.io/?url=https://raw.githubusercontent.com/samsung-cnct/cma-ssh/master/assets/generated/swagger/api.swagger.json)


### Requirements
- Kubernetes 1.10+

### Deployment
The default way to deploy CMA-SSH is by the provided helm chart located in the `deployment/helm/cma-ssh` directory.

#### install via [helm](https://helm.sh/docs/using_helm/#quickstart)
1. Locate the private IP of a k8s node that cma-ssh is going to be deployed on and will be used as the `install.bootstrapIp`.
1. Locate the nginx proxy used by the airgap environment to be used as the  `install.airgapProxyIp`.
1. Install helm chart passing in the above values:
    ```bash
    helm install deployments/helm/cma-ssh --name cma-ssh --set install.bootstrapIp="ip from step 1" --set install.airgapProxyIp="ip of step 2"
    ```
    *alternatively you can update `values.yaml` with IPs

### Utilizes:
- [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder)
- [Protocol Buffers](https://developers.google.com/protocol-buffers)
- [kustomize]()

## Build 
#### one time setup of tools
- mac osx: 
`make -f build/Makefile install-tools-darwin`

- linux:
`make -f build/Makefile install-tools-linux`

#### To generate code and binary:
- mac osx: 
`make -f build/Makefile darwin`

- linux:
`make -f build/Makefile linux`

CRDs are generated in `./crd`  
RBAC is generated in `./rbac`

Helm chart under `./deployments/helm/cma-ssh` gets updated with the right CRDs and RBAC

## Testing with Azure

Requirements:
- docker
- [opctl](https://opctl.io/docs/getting-started/opctl.html)
- [azure cli](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli?view=azure-cli-latest)
    - install on mac osx: `brew install azure-cli`

#### Setup steps:
1. create the ssh key pair (requires rsa and 2048 bit)
    **no password**
    ```bash
    ssh-keygen -t rsa -b 2048 -f id_rsa
    ```
    
2. create `args.yml` file
    ```bash
    touch .opspec/args.yml
    ```
    add inputs:
    ```$xslt
      subscriptionId: <azure subscription id>
      loginId: <azure service principal id (must have permission to edit user permissions in subscription>
      loginSecret: <azure service principal secret>
      loginTenantId: <azure active directory id>
      sshKeyValue: <path to public key from step 1>
      sshPrivateKey: <path to private key from step 1>
      clusterAccountId: <azure service principal for in cluster resources (ex: load balancer creation)>
      clusterAccountSecret: <azure service principal secret>
      rootPassword: <root password for client vm>
      name: <prefix name to give to all resources> (ex: zaptest01)
    ```

3. from root directory of repo run 
    ```bash
    opctl run build-azure
    ```
    first run takes 10/15 minutes. *this can be run multiple times
    
4. to get kubeconfig for central cluster:
    - login to azure via cli: 
        ```bash
        az login
        ```
    - get kubeconfig from aks cluster: 
        ```bash
        az aks get-credentials -n <name> -g <name>-group
        ```
        *replace with name from args.yml (step 2)

5. install bootstrap and connect to proxy: 
    ```bash
    helm install deployments/helm/cma-ssh --name cma-ssh \
    --set install.operator=false \
    --set images.bootstrap.tag=0.1.17-local \
    --set install.bootstrapIp=10.240.0.6 \
    --set install.airgapProxyIp=10.240.0.7
    ```
    * check bootstrap latest tag at [quay.io](https://quay.io/repository/samsung_cnct/cma-ssh-bootstrap?tab=tags)
    * bootstrapIP is any node private ip (most likely: 10.240.0.4 thru .6)
    * to get airgapProxyIp run:
    ```bash
    az vm show -g <name>-group -n <name>-proxy -d --query publicIps --out tsv
    ``` 
6. locally start operator 
    ```bash
    CMA_BOOTSTRAP_IP=10.240.0.6 CMA_NEXUS_PROXY_IP=10.240.0.7 ./cma-ssh
    ```

#### creating additional azure vm for testing clusters:
* to create additional vms:
```bash
opctl run create-vm
```
* this will create a new vm and provide the name/public ip

* TODO: return private IP also

#### cleanup azure:
* TODO: create azure-delete op.

* currently requires manually deleting resources / resource group manually in the azure portal or cli

* resource group will be named `<name>-group` from `args.yml` file.

[generate-an-api-key]: https://docs.maas.io/2.1/en/manage-account#api-key
