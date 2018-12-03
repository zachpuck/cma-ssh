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

echo "checking for existing vnet"
if [ "$(az network vnet show --resource-group "$resourceGroup" --name "$name-vnet")" != "" ]
then
  echo "found exiting vnet"
  az network vnet subnet show -g $resourceGroup --vnet-name $name-vnet -n $name-subnet --query id --output tsv > /subnetID
else
    echo "creating vnet"
    az network vnet create -n $name-vnet \
    -g $resourceGroup \
    --address-prefix 10.0.0.0/8 \
    --subnet-name $name-subnet \
    --subnet-prefix 10.240.0.0/16 > /dev/null

    az network vnet subnet show -g $resourceGroup --vnet-name $name-vnet -n $name-subnet --query id --output tsv > /subnetID
fi
