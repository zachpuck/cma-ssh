#!/usr/bin/env sh

### begin login
loginCmd='az login -u "$loginId" -p "$loginSecret"'

# handle opts
if [ "$loginTenantId" != " " ]; then
    loginCmd=$(printf "%s --tenant %s" "$loginCmd" "$loginTenantId")
fi

case "$loginType" in
    "user")
        echo "logging in as user"
        ;;
    "sp")
        echo "logging in as service principal"
        loginCmd=$(printf "%s --service-principal" "$loginCmd")
        ;;
esac
eval "$loginCmd" >/dev/null

echo "setting default subscription"
az account set --subscription "$subscriptionId"
### end login

echo "checking for existing nginx proxy"
if [ "$(az vm show --resource-group "$resourceGroup" --name "$name-proxy")" != "" ]
then
  echo "found exiting nginx proxy"
  az vm show -g $resourceGroup -n $name-proxy -d --query publicIps --out tsv > /nginxIP
else
    echo "creating nginx proxy vm"
    az vm create -n $name-proxy \
    -g $resourceGroup \
    --image $image \
    --admin-username $adminUsername \
    --ssh-key-value /sshKeyValue \
    --vnet-name $name-vnet \
    --subnet $name-subnet \
    --query "publicIpAddress" --output tsv > /nginxIP
fi

echo "checking for existing control plane"
if [ "$(az vm show --resource-group "$resourceGroup" --name "$name-cp")" != "" ]
then
  echo "found exiting control plane"
  az vm show -g $resourceGroup -n $name-cp -d --query publicIps --out tsv > /controlPlaneIP
else
    echo "creating control plane vm"
    az vm create -n $name-cp \
    -g $resourceGroup \
    --image $image \
    --admin-username $adminUsername \
    --ssh-key-value /sshKeyValue \
    --vnet-name $name-vnet \
    --subnet $name-subnet \
    --query "publicIpAddress" --output tsv > /controlPlaneIP

    echo "disabling internet access on control plane"
    az network nsg rule create -n DenyInternetAccess -g $resourceGroup \
        --nsg-name $name-cpNSG  --priority 4096 \
        --source-address-prefixes VirtualNetwork --source-port-ranges '*' \
        --destination-address-prefixes Internet --destination-port-ranges '*' \
        --direction Outbound --access Deny > /dev/null
fi

echo "checking for existing node"
if [ "$(az vm show --resource-group "$resourceGroup" --name "$name-node")" != "" ]
then
  echo "found exiting node"
  az vm show -g $resourceGroup -n $name-node -d --query publicIps --out tsv > /nodeIP
else
    echo "creating node vm"
    az vm create -n $name-node \
    -g $resourceGroup \
    --image $image \
    --admin-username $adminUsername \
    --ssh-key-value /sshKeyValue \
    --vnet-name $name-vnet \
    --subnet $name-subnet \
    --query "publicIpAddress" --output tsv > /nodeIP

    echo "disabling internet access on node"
    az network nsg rule create -n DenyInternetAccess -g $resourceGroup \
        --nsg-name $name-nodeNSG  --priority 4096 \
        --source-address-prefixes VirtualNetwork --source-port-ranges '*' \
        --destination-address-prefixes Internet --destination-port-ranges '*' \
        --direction Outbound --access Deny > /dev/null
fi

#echo "opening 443 port on NSG for kubernetes api"
#az network nsg rule create -n kubernetesAPIAccess -g $resourceGroup --nsg-name $name-cpNSG  --priority 101 --destination-port-ranges 443 > /dev/null

# TODO: remove SSH inbound ports from control plane and node
