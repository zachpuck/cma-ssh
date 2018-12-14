# Build the manager binary
FROM golang:1.10.3 as builder

# Copy in the go src
WORKDIR /go/src/github.com/samsung-cnct/cma-ssh
COPY .  ./


# Build
ENV KUBEBUILDER_VERSION=1.0.5
ENV KUBEBUILDER_ARCH=amd64
ENV KUSTOMIZE_VERSION=1.0.10
ENV KUSTOMIZE_ARCH=amd64
ENV PROTOC_VERSION=3.6.1
ENV PROTOC_LINUX_FILENAME=protoc-${PROTOC_VERSION}-linux-x86_64.zip
ENV PROTOC_LINUX_DOWNLOAD_URL=https://github.com/google/protobuf/releases/download/v${PROTOC_VERSION}/${PROTOC_LINUX_FILENAME}
ENV PATH="${PATH}:/usr/local/kubebuilder/bin"
RUN apt-get update -y \
    && apt-get install unzip -y \
    && curl -L -O https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${KUBEBUILDER_VERSION}/kubebuilder_${KUBEBUILDER_VERSION}_linux_${KUBEBUILDER_ARCH}.tar.gz \
    && tar -zxvf kubebuilder_${KUBEBUILDER_VERSION}_linux_${KUBEBUILDER_ARCH}.tar.gz \
    && mv kubebuilder_${KUBEBUILDER_VERSION}_linux_${KUBEBUILDER_ARCH} /usr/local/kubebuilder \
    && curl -L -O https://github.com/kubernetes-sigs/kustomize/releases/download/v${KUSTOMIZE_VERSION}/kustomize_${KUSTOMIZE_VERSION}_linux_${KUSTOMIZE_ARCH} \
    && mv kustomize_${KUSTOMIZE_VERSION}_linux_${KUSTOMIZE_ARCH} /usr/local/bin/kustomize \
    && chmod +x /usr/local/bin/kustomize \
    && curl -L -O ${PROTOC_LINUX_DOWNLOAD_URL} \
    && unzip -o ${PROTOC_LINUX_FILENAME} -d /usr/local/
RUN make -f build/Makefile install-dep linux

# Copy the controller-manager into a thin image
FROM alpine:3.8
WORKDIR /root/
COPY --from=builder /go/src/github.com/samsung-cnct/cma-ssh/cma-ssh .
ENTRYPOINT ["./cma-ssh"]
