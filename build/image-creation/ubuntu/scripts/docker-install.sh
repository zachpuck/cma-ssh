apt-get update -y
apt-get install -y \
  apt-transport-https \
  ca-certificates \
  curl \
  gnupg-agent \
  software-properties-common \
  build-essentials \
  linux-headers-$(uname -r)

# https://docs.nvidia.com/cuda/cuda-installation-guide-linux/index.html#pre-installation-actions
# https://github.com/NVIDIA/nvidia-docker/wiki/Frequently-Asked-Questions#how-do-i-install-20-if-im-not-using-the-latest-docker-version
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -
apt-key adv --fetch-keys https://developer.download.nvidia.com/compute/cuda/repos/18.04/amd64/7fa2af80.pub

# The nvidia cards on the SDSA lab GPU hosts are
# 04:00.0 3D controller: NVIDIA Corporation GK210GL [Tesla K80] (rev a1)
# which are CUDA
add-apt-repository \
  "deb https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) \
  stable"
apt-get update -y
apt-get install -y nvidia-docker2=2.0.2 \
                   nvidia-container-runtime=2.0.2 \
                   docker-ce docker-ce-cli \
                   containerd.io cuda
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
