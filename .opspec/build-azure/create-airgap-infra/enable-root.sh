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

echo "get control plane public IP"
cpIP=$(az vm show -g $resourceGroup -n $name-cp -d --query publicIps --out tsv)

echo "enable ssh as root on control plane"
ssh -o "StrictHostKeyChecking no" -i /privateKey centos@$cpIP \
    'export TERM=xterm;
    sudo sed -i s/^#PermitRootLogin\ yes/PermitRootLogin\ without-password/ /etc/ssh/sshd_config;
    sudo mkdir /root/.ssh;
    sudo cp ~/.ssh/authorized_keys /root/.ssh'

echo "get node public IP"
nodeIP=$(az vm show -g $resourceGroup -n $name-node -d --query publicIps --out tsv)

echo "enable ssh as root on node"
ssh -o "StrictHostKeyChecking no" -i /privateKey centos@$nodeIP \
    'export TERM=xterm;
    sudo sed -i s/^#PermitRootLogin\ yes/PermitRootLogin\ without-password/ /etc/ssh/sshd_config;
    sudo mkdir /root/.ssh;
    sudo cp ~/.ssh/authorized_keys /root/.ssh'
