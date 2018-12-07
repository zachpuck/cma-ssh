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

guid=${uuid:0:6}

echo "Machine name: ${name}-${guid}"

echo "creating vm"
az vm create -n ${name}-${guid} \
-g $resourceGroup \
--image $image \
--admin-username $adminUsername \
--ssh-key-value /sshKeyValue \
--vnet-name $name-vnet \
--subnet $name-subnet \
--query "publicIpAddress" --output tsv

echo "disabling internet access on vm"
az network nsg rule create -n DenyInternetAccess -g $resourceGroup \
    --nsg-name ${name}-${guid}NSG  --priority 4096 \
    --source-address-prefixes VirtualNetwork --source-port-ranges '*' \
    --destination-address-prefixes Internet --destination-port-ranges '*' \
    --direction Outbound --access Deny > /dev/null
