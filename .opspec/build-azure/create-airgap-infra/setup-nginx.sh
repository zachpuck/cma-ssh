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

echo "get nginx public IP"
nginxIP=$(az vm show -g $resourceGroup -n $name-proxy -d --query publicIps --out tsv)

echo "install nginx"
ssh -o "StrictHostKeyChecking no" -i /privateKey centos@$nginxIP \
    'export TERM=xterm;
    sudo sed -i /etc/selinux/config -r -e "s/^SELINUX=.*/SELINUX=disabled/g";
    sudo setenforce 0;
    sudo yum install -y epel-release;
    sudo yum install -y nginx'

echo "copy nginx-configs config files to proxy"
scp -o "StrictHostKeyChecking no" -i /privateKey -r /nginx-configs centos@$nginxIP:

echo "start nginx service"
ssh -o "StrictHostKeyChecking no" -i /privateKey centos@$nginxIP \
    'export TERM=xterm;
    sudo mv ~/nginx-configs/nginx.conf /etc/nginx;
    sudo mv ~/nginx-configs/* /etc/nginx/conf.d;
    sudo chown root:root /etc/nginx/nginx.conf;
    sudo chown root:root /etc/nginx/conf.d/*;
    sudo systemctl start nginx'
