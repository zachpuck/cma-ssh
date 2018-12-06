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
ENV PATH="${PATH}:/usr/local/kubebuilder/bin"
RUN curl -L -O https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${KUBEBUILDER_VERSION}/kubebuilder_${KUBEBUILDER_VERSION}_linux_${KUBEBUILDER_ARCH}.tar.gz \
    && tar -zxvf kubebuilder_${KUBEBUILDER_VERSION}_linux_${KUBEBUILDER_ARCH}.tar.gz \
    && mv kubebuilder_${KUBEBUILDER_VERSION}_linux_${KUBEBUILDER_ARCH} /usr/local/kubebuilder \
    && curl -L -O https://github.com/kubernetes-sigs/kustomize/releases/download/v${KUSTOMIZE_VERSION}/kustomize_${KUSTOMIZE_VERSION}_linux_${KUSTOMIZE_ARCH} \
    && mv kustomize_${KUSTOMIZE_VERSION}_linux_${KUSTOMIZE_ARCH} /usr/local/bin/kustomize \
    && chmod +x /usr/local/bin/kustomize
RUN make -f build/Makefile install-dep linux

# Copy the controller-manager into a thin image
FROM alpine:3.8
WORKDIR /root/
COPY --from=builder /go/src/github.com/samsung-cnct/cma-ssh/cma-ssh .
ENTRYPOINT ["./cma-ssh"]