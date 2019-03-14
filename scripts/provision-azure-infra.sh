#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

[[ ${DEBUG:-1} ]] && set -o xtrace

__dir="$(cd -P -- "$(dirname -- "$0")" && cd ../ && pwd -P)"

# This script expects the environment to be CentOS

export AZURE_LOCATION="${AZURE_LOCATION:-westus}"
export ADMIN_USER_NAME="${ADMIN_USER_NAME:-centos}"
export VM_IMAGE="${VM_IMAGE:-OpenLogic:CentOS:7.4:latest}"
export ROOT_PASSWORD="${ROOT_PASSWORD:-a2dZVs8k4zYeLde3!}"
export NAME_PREFIX="${NAME_PREFIX:-delete-me}"
export K8S_VERSION="${K8S_VERSION:-1.10.6}"
export AKS_K8S_VERSION="${AKS_K8S_VERSION:-1.12.6}"
export ROOT_PASSWORD=${ROOT_PASSWORD:-$(date +%c | md5sum | base64)}
export KEY_HOME=${KEY_HOME:-$HOME/.ssh}
export KEY_NAME=${KEY_NAME:-${KEY_HOME}/sshKey}
export SSH_PRIVATE_KEY="${KEY_NAME}"
export SSH_PUBLIC_KEY="$SSH_PRIVATE_KEY.pub"
export resource_group="${resource_group:-$NAME_PREFIX-group}"

mkdir -p "$KEY_HOME"

# requires RSA and 2048 bit keys only with no password: "ssh-keygen -t rsa -b 2048 -f id_rsa"
# if ssh-keygen is not found, it will apk add it.
if ! which ssh-keygen >/dev/null 2>&1; then apk -U add openssh; fi

# necessary for local testing
rm -rf "$SSH_PRIVATE_KEY" >/dev/null 2>&1 || true
ssh-keygen -t rsa -b 2048 -f "$SSH_PRIVATE_KEY" -N ''

echo "creating resource group"
az group create --name "${resource_group}" --location "${AZURE_LOCATION}"

echo "creating virtual network (vnet)"
az network vnet create -n "${NAME_PREFIX}"-vnet \
    -g "${resource_group}" \
    --address-prefix 10.0.0.0/8 \
    --subnet-name "${NAME_PREFIX}"-subnet \
    --subnet-prefix 10.240.0.0/16

vnetSubnetID=$(az network vnet subnet show -g "${resource_group}" --vnet-name "${NAME_PREFIX}"-vnet -n "${NAME_PREFIX}"-subnet --query id --output tsv)

echo "setting clusterAccountId as owner of resource Group due to custom vnet"
az role assignment create --assignee "${AZURE_CLIENT_ID}" \
    --role Owner \
    --scope /subscriptions/"${AZURE_SUBSCRIPTION_ID}"/resourceGroups/"${resource_group}"

### AKS cluster setup as CMC ###
{
  echo "creating aks cluster"
  az aks create -n "${NAME_PREFIX}" -g "${resource_group}" \
      --ssh-key-value "${SSH_PUBLIC_KEY}" \
      --kubernetes-version "${AKS_K8S_VERSION}" \
      --service-principal "${AZURE_CLIENT_ID}" \
      --client-secret "${AZURE_CLIENT_SECRET}" \
      --vnet-subnet-id "$vnetSubnetID"

  aksMachineResourceGroup=MC_"${resource_group}"_"${NAME_PREFIX}"_"${AZURE_LOCATION}"

  aksRouteTable=$(az network route-table list -g "$aksMachineResourceGroup" --query [].id --output tsv)
  aksNSG_id=$(az network nsg list -g "$aksMachineResourceGroup" --query [].id --output tsv)
  aksNSG_name=$(az network nsg list -g "$aksMachineResourceGroup" --query [].name --output tsv)

  echo "update custom vnet"
  az network vnet subnet update \
      --route-table "$aksRouteTable" \
      --network-security-group "$aksNSG_id" \
      --ids "$vnetSubnetID"

  echo "enable ssh on aks nsg"
  az network nsg rule create -n EnableSSH -g "$aksMachineResourceGroup" \
      --nsg-name "$aksNSG_name" --priority 1000 \
      --destination-port-ranges 22

  echo "set kubeconfig to aks cluster"
  az aks get-credentials -n "${NAME_PREFIX}" -g "${resource_group}"
} &

aks_pid=$!
### END AKS SETUP ###

#### PROXY VM ####
{
  echo "creating proxy vm"
  az vm create -n "${NAME_PREFIX}"-proxy \
      -g "${resource_group}" \
      --image "${VM_IMAGE}" \
      --admin-username "${ADMIN_USER_NAME}" \
      --ssh-key-value "${SSH_PUBLIC_KEY}" \
      --vnet-name "${NAME_PREFIX}"-vnet \
      --subnet "${NAME_PREFIX}"-subnet

  echo "get proxy public IP"
  pubProxyIP=$(az vm show -g "${resource_group}" -n "${NAME_PREFIX}"-proxy -d --query publicIps --out tsv)
  proxyIP=$(az vm show -g "${resource_group}" -n "${NAME_PREFIX}"-proxy -d --query privateIps --out tsv)
  echo "$proxyIP" > proxyIP

  echo "install docker"
  # shellcheck disable=SC2086
  ssh -o "StrictHostKeyChecking no" -i "${SSH_PRIVATE_KEY}" centos@$pubProxyIP \
      'export TERM=xterm;
      sudo sed -i /etc/selinux/config -r -e "s/^SELINUX=.*/SELINUX=disabled/g";
      sudo setenforce 0;
      sudo yum install -y docker;
      sudo systemctl start docker'

  echo "copy docker-configs config files to proxy"
  # shellcheck disable=SC2086
  scp -o "StrictHostKeyChecking no" -i "${SSH_PRIVATE_KEY}" -r "$__dir"/scripts/docker-configs centos@$pubProxyIP:

  echo "start docker containers"
  # shellcheck disable=SC2086
  ssh -o "StrictHostKeyChecking no" -i "${SSH_PRIVATE_KEY}" centos@$pubProxyIP \
  ssh -o "StrictHostKeyChecking no" -i "${SSH_PRIVATE_KEY}" centos@$pubProxyIP \
      'export TERM=xterm;
      sudo docker kill dockerhub-proxy > /dev/null && sudo docker rm dockerhub-proxy  > /dev/null;
      sudo docker kill gcr-proxy > /dev/null && sudo docker rm gcr-proxy  > /dev/null;
      sudo docker kill k8sgcr-proxy > /dev/null && sudo docker rm k8sgcr-proxy  > /dev/null;
      sudo docker kill quay-proxy > /dev/null && sudo docker rm quay-proxy  > /dev/null;
      sudo docker run -d --restart=always -p 9401:5000 --name dockerhub-proxy -v $(pwd)/docker-configs/config-dockerhub.yaml:/etc/docker/registry/config.yml registry:2.6.2;
      sudo docker run -d --restart=always -p 9402:5000 --name gcr-proxy -v $(pwd)/docker-configs/config-gcr.yaml:/etc/docker/registry/config.yml registry:2.6.2;
      sudo docker run -d --restart=always -p 9403:5000 --name k8sgcr-proxy -v $(pwd)/docker-configs/config-k8sgcr.yaml:/etc/docker/registry/config.yml registry:2.6.2;
      sudo docker run -d --restart=always -p 9404:5000 --name quay-proxy -v $(pwd)/docker-configs/config-quay.yaml:/etc/docker/registry/config.yml registry:2.6.2;'
} &

proxy_pid=$!
### END PROXY VM ####
# kick off creation of the worker nodes in parallel!

for vm in {0..1}; do
  export VM_PREFIX=vm-$((vm + 1))

  "$__dir"/scripts/create-vm &

  pids[$vm]=$!:$VM_PREFIX
done

for ((c=0; c < ${#pids[@]} ;c++)); do
  echo "c:pid: $c:${pids[$c]}"

  pid_pair="$c:${pids[$c]}"
  IFS=":" read -r c pid VM_PREFIX <<< "$pid_pair"

  # shellcheck disable=SC2086
  wait $pid

  case $c in
    0)
      CONTROL_PLANE_PRIVATE_IP=$(az vm show -g "${resource_group}" -n "${NAME_PREFIX}"-"${VM_PREFIX}" -d --query privateIps --out tsv)
      CONTROL_PLANE_PUBLIC_IP=$(az vm show -g "${resource_group}" -n "${NAME_PREFIX}"-"${VM_PREFIX}" -d --query publicIps --out tsv)
      ;;
    1)
      WORKER01_PRIVATE_IP=$(az vm show -g "${resource_group}" -n "${NAME_PREFIX}"-"${VM_PREFIX}" -d --query privateIps --out tsv)
      WORKER01_PUBLIC_IP=$(az vm show -g "${resource_group}" -n "${NAME_PREFIX}"-"${VM_PREFIX}" -d --query publicIps --out tsv)
      ;;
  esac
done

echo "$CONTROL_PLANE_PRIVATE_IP" > CONTROL_PLANE_PRIVATE_IP
echo "$CONTROL_PLANE_PUBLIC_IP"  > CONTROL_PLANE_PUBLIC_IP
echo "$WORKER01_PRIVATE_IP"      > WORKER01_PRIVATE_IP
echo "$WORKER01_PUBLIC_IP"       > WORKER01_PUBLIC_IP

wait $aks_pid   || true
wait $proxy_pid || true
##### END ####
