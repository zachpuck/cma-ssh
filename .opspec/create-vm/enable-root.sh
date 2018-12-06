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

echo "get vm public IP"
vmIP=$(az vm show -g $resourceGroup -n ${name}-${guid} -d --query publicIps --out tsv)

echo "enable ssh as root on vm"
ssh -o "StrictHostKeyChecking no" -i /privateKey centos@$vmIP \
    'export TERM=xterm;
    sudo sed -i s/^#PermitRootLogin\ yes/PermitRootLogin\ without-password/ /etc/ssh/sshd_config;
    sudo mkdir /root/.ssh;
    sudo cp ~/.ssh/authorized_keys /root/.ssh'
