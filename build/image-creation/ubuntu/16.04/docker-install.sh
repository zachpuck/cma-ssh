
DEBIAN_FRONTEND=noninteractive
export DEBIAN_FRONTEND

apt-get install -y --no-install-recommends --no-install-suggests \
  apt-transport-https \
  ca-certificates \
  curl \
  gnupg-agent \
  software-properties-common

curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -

add-apt-repository \
  "deb https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) \
  stable"
apt-get update
apt-get install -y --no-install-recommends --no-install-suggests \
                   docker-ce docker-ce-cli \
                   containerd.io
sudo pkill -SIGHUP dockerd
apt --purge autoremove -y
apt autoclean -y
apt clean -y

mkdir -p /etc/systemd/system/docker.service.d
mkdir /etc/docker
cat <<EOF > /etc/docker/daemon.json
{
  "exec-opts": ["native.cgroupdriver=systemd"],
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m"
  },
  "storage-driver": "overlay2"
}
EOF
systemctl enable docker
