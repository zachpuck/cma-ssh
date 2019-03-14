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

echo "checking for existing proxy"
if [ "$(az vm show --resource-group "$resourceGroup" --name "$name-proxy")" != "" ]
then
  echo "found exiting proxy"
  az vm show -g $resourceGroup -n $name-proxy -d --query publicIps --out tsv > /nginxIP
else
    echo "creating proxy vm"
    az vm create -n $name-proxy \
    -g $resourceGroup \
    --image $image \
    --admin-username $adminUsername \
    --ssh-key-value /sshKeyValue \
    --vnet-name $name-vnet \
    --subnet $name-subnet \
    --query "publicIpAddress" --output tsv > /nginxIP
fi
