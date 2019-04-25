mkdir -p /etc/apt/sources.list.d
add-apt-repository -y ppa:graphics-drivers/ppa

# Add the package repositories
curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | \
  sudo apt-key add -
distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
curl -s -L https://nvidia.github.io/nvidia-docker/$distribution/nvidia-docker.list | \
  sudo tee /etc/apt/sources.list.d/nvidia-docker.list
sudo apt-get update

# Install nvidia-docker2 and reload the Docker daemon configuration
apt-get install -y \
  apt-transport-https \
  ca-certificates \
  curl \
  gnupg-agent \
  software-properties-common

# https://docs.nvidia.com/cuda/cuda-installation-guide-linux/index.html#pre-installation-actions
# https://github.com/NVIDIA/nvidia-docker/wiki/Frequently-Asked-Questions#how-do-i-install-20-if-im-not-using-the-latest-docker-version
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -

# The nvidia cards on the SDSA lab GPU hosts are
# 04:00.0 3D controller: NVIDIA Corporation GK210GL [Tesla K80] (rev a1)
# which are CUDA
add-apt-repository \
  "deb https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) \
  stable"
apt-get update
apt-get install -y nvidia-docker2 \
                   docker-ce docker-ce-cli \
                   containerd.io libcuda1-384 \
                   nvidia-384
sudo pkill -SIGHUP dockerd
apt --purge autoremove -y
apt autoclean -y
apt clean -y

mkdir -p /etc/systemd/system/docker.service.d
mkdir /etc/docker
cat <<EOF > /etc/docker/daemon.json
{
  "default-runtime": "nvidia",
  "runtimes": {
    "nvidia": {
      "path": "/usr/bin/nvidia-container-runtime",
      "runtimeArgs": []
    }
  }
  "exec-opts": ["native.cgroupdriver=systemd"],
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m"
  },
  "storage-driver": "overlay2"
}
EOF
systemctl enable docker
