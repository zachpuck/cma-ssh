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

echo "checking for existing aks cluster"
if [ "$(az aks show --resource-group "$resourceGroup" --name "$name")" != "" ]
then
  echo "found exiting aks cluster"
else
    echo "setting clusterAccountId as owner of resource Group due to custom vnet"
    az role assignment create --assignee $clusterAccountId \
    --role Owner \
    --scope /subscriptions/$subscriptionId/resourceGroups/$resourceGroup > /dev/null

    echo "creating aks cluster"
    az aks create -n $name -g $resourceGroup \
    --ssh-key-value /sshKeyValue \
    --kubernetes-version $k8sVersion \
    --service-principal $clusterAccountId \
    --client-secret $clusterAccountSecret \
    --vnet-subnet-id $vnetSubnetID > /dev/null

    aksMachineResourceGroup="MC_${resourceGroup}_${name}_${location}"

    aksRouteTable=$(az network route-table list -g $aksMachineResourceGroup --query [].id --output tsv)
    aksNSG_id=$(az network nsg list -g $aksMachineResourceGroup --query [].id --output tsv)
    aksNSG_name=$(az network nsg list -g $aksMachineResourceGroup --query [].name --output tsv)

    echo "update custom vnet"
    az network vnet subnet update \
        --route-table $aksRouteTable \
        --network-security-group $aksNSG_id \
        --ids $vnetSubnetID > /dev/null

    echo "enable ssh on aks nsg"
    az network nsg rule create -n EnableSSH -g $aksMachineResourceGroup \
        --nsg-name $aksNSG_name --priority 1000 \
        --destination-port-ranges 22 > /dev/null
fi
