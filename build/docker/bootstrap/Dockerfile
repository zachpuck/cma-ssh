FROM centos:7.4.1708 as downloader

COPY clean-install.sh /usr/local/bin/clean-install
RUN chmod +x /usr/local/bin/clean-install

# Use clean-install script to install packages and clean up afterwords
RUN clean-install epel-release yum-utils wget

# all baked kubernetes versions
# TODO: set this through make
ARG K8S_VERSIONS="1.12.6 1.13.4"
# helm versions
ARG HELM_VERSIONS="v2.11.0"
# docker version
# TODO: set this through make
ARG DOCKER_VERSION="18.06.2.ce"
# flannel version
ARG FLANNEL_VERSION="a70459be0084506e4ec919aa1c114638878db11b"

# add docker-ce repo, add kubernetes repo
RUN yum-config-manager \
    --add-repo \
    https://download.docker.com/linux/centos/docker-ce.repo \
    && yum-config-manager --add-repo https://packages.cloud.google.com/yum/repos/kubernetes-el7-x86_64 \
    && yum update -y

# pre create folders
RUN mkdir -p /var/tmp/yaml \
    && mkdir -p /var/tmp/bin \
    && mkdir -p /var/tmp/rpm

# donwload all k8s rpms and helm
RUN helmversions=($HELM_VERSIONS); k8sversions=($K8S_VERSIONS); for ver in ${versions[*]}; do mkdir -p /var/tmp/${ver}; done \
    && for ver in ${k8sversions[*]}; do yumdownloader --assumeyes --resolve --destdir /var/tmp/rpm/k8s-${ver} \
        kubectl-${ver} kubeadm-${ver} kubelet-${ver} kubernetes-cni; done \
    && yumdownloader --assumeyes --resolve --destdir /var/tmp/rpm/docker audit audit-libs libselinux libsepol libsemanage \
        device-mapper-persistent-data lvm2  conntrack docker-ce-${DOCKER_VERSION} \
    && yumdownloader --assumeyes --resolve --destdir /var/tmp/rpm/nginx nginx \
    && yumdownloader --assumeyes --resolve --destdir /var/tmp/rpm/curl curl \
    && yumdownloader --assumeyes --resolve --destdir /var/tmp/rpm/wget wget \
    && wget --directory-prefix=/var/tmp/yaml https://raw.githubusercontent.com/coreos/flannel/${FLANNEL_VERSION}/Documentation/kube-flannel.yml \
    && for ver in ${helmversions[*]}; do wget --directory-prefix=/var/tmp/bin/${ver} https://storage.googleapis.com/kubernetes-helm/helm-${ver}-linux-amd64.tar.gz; done

# application image
FROM centos:7.4.1708

RUN yum -y install epel-release && \
    yum -y update && \
    yum -y install createrepo yum-utils nginx && \
    yum clean all

RUN mkdir /yum && \
    chmod 777 /yum && \
    mkdir -p /static && \
    chmod 777 /static

COPY nginx.conf /etc/nginx/nginx.conf
COPY --from=downloader /var/tmp/rpm/ /yum
COPY --from=downloader /var/tmp/yaml/ /static
COPY --from=downloader /var/tmp/bin/ /static

COPY entrypoint.sh /entrypoint.sh
RUN chmod 700 /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]