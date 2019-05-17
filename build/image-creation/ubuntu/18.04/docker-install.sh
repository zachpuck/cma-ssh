apt-get update -y
apt-get install -y \
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
apt-get update -y
apt-get install -y docker-ce docker-ce-cli \
                   containerd.io
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
