FROM centos:7 as builder

# setup sudo and other tools (for testing only)
RUN yum update -y \
    && yum install epel-release -y \
    && yum install yum-utils wget -y \
    && yum clean all

# all baked kubernetes versions
ARG K8S_VERSIONS="1.10.6 1.11.2"
# helm version
ARG HELM_VERSION=v2.11.0

# add kubernetes repo
RUN echo -e '[kubernetes]\n\
name=Kubernetes\n\
baseurl=https://packages.cloud.google.com/yum/repos/kubernetes-el7-x86_64\n\
enabled=1\n\
gpgcheck=1\n\
repo_gpgcheck=0'\
>> /etc/yum.repos.d/kubernetes.repo

# pre create folders
RUN mkdir -p /var/tmp/yaml \
    && mkdir -p /var/tmp/bin \
    && mkdir -p /var/tmp/rpm

# donwload all k8s rpms and helm
RUN versions=($K8S_VERSIONS); for ver in ${versions[*]}; do mkdir -p /var/tmp/${ver}; done \
    && for ver in ${versions[*]}; do yumdownloader --assumeyes --resolve --destdir /var/tmp/rpm/${ver} kubectl-${ver}; done \
    && for ver in ${versions[*]}; do yumdownloader --assumeyes --resolve --destdir /var/tmp/rpm/${ver} kubeadm-${ver}; done \
    && for ver in ${versions[*]}; do yumdownloader --assumeyes --resolve --destdir /var/tmp/rpm/${ver} kubelet-${ver}; done \
    && for ver in ${versions[*]}; do yumdownloader --assumeyes --resolve --destdir /var/tmp/rpm/${ver} kubernetes-cni; done \
    && wget --directory-prefix=/var/tmp/yaml https://raw.githubusercontent.com/coreos/flannel/v0.10.0/Documentation/kube-flannel.yml \
    && wget --directory-prefix=/var/tmp/bin https://storage.googleapis.com/kubernetes-helm/helm-${HELM_VERSION}-linux-amd64.tar.gz

FROM centos:7.4.1708

RUN mkdir -p /resources/yaml \
    && mkdir -p /resources/rpms \
    && mkdir -p /resources/bin

COPY --from=builder /var/tmp/rpm /resources/rpms
COPY --from=builder /var/tmp/yaml /resources/yaml
COPY --from=builder /var/tmp/bin /resources/bin
