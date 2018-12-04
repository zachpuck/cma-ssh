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

echo "get proxy public IP"
proxyIP=$(az vm show -g $resourceGroup -n $name-proxy -d --query publicIps --out tsv)

echo "install docker"
ssh -o "StrictHostKeyChecking no" -i /privateKey centos@$proxyIP \
    'export TERM=xterm;
    sudo sed -i /etc/selinux/config -r -e "s/^SELINUX=.*/SELINUX=disabled/g";
    sudo setenforce 0;
    sudo yum install -y docker;
    sudo systemctl start docker'

echo "copy docker-configs config files to proxy"
scp -o "StrictHostKeyChecking no" -i /privateKey -r /docker-configs centos@$proxyIP:

echo "start docker containers"
ssh -o "StrictHostKeyChecking no" -i /privateKey centos@$proxyIP \
    'export TERM=xterm;
    sudo docker kill dockerhub-proxy > /dev/null && sudo docker rm dockerhub-proxy  > /dev/null;
    sudo docker kill gcr-proxy > /dev/null && sudo docker rm gcr-proxy  > /dev/null;
    sudo docker kill k8sgcr-proxy > /dev/null && sudo docker rm k8sgcr-proxy  > /dev/null;
    sudo docker kill quay-proxy > /dev/null && sudo docker rm quay-proxy  > /dev/null;
    sudo docker run -d --restart=always -p 9401:5000 --name dockerhub-proxy -v $(pwd)/docker-configs/config-dockerhub.yaml:/etc/docker/registry/config.yml registry:2;
    sudo docker run -d --restart=always -p 9402:5000 --name gcr-proxy -v $(pwd)/docker-configs/config-gcr.yaml:/etc/docker/registry/config.yml registry:2;
    sudo docker run -d --restart=always -p 9403:5000 --name k8sgcr-proxy -v $(pwd)/docker-configs/config-k8sgcr.yaml:/etc/docker/registry/config.yml registry:2;
    sudo docker run -d --restart=always -p 9404:5000 --name quay-proxy -v $(pwd)/docker-configs/config-quay.yaml:/etc/docker/registry/config.yml registry:2;'


