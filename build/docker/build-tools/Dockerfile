FROM golang:1.12-stretch

RUN apt update -y && apt install unzip -y

# kubebuilder requires dep
RUN go get -u github.com/golang/dep/cmd/dep

# kubebuilder requires kustomize
ARG OS=linux
ARG ARCH=amd64
ARG KUSTOMIZE_VERSION=2.0.3
RUN curl -L https://github.com/kubernetes-sigs/kustomize/releases/download/v${KUSTOMIZE_VERSION}/kustomize_${KUSTOMIZE_VERSION}_${OS}_${ARCH} \
	-o /usr/local/bin/kustomize

# install kubebuilder
ARG KUBEBUILDER_VERSION=1.0.8
RUN curl -L -O "https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${KUBEBUILDER_VERSION}/kubebuilder_${KUBEBUILDER_VERSION}_${OS}_${ARCH}.tar.gz"
RUN tar -zxvf kubebuilder_${KUBEBUILDER_VERSION}_${OS}_${ARCH}.tar.gz
RUN mv kubebuilder_${KUBEBUILDER_VERSION}_${OS}_${ARCH} /usr/local/kubebuilder
ENV PATH=$PATH:/usr/local/kubebuilder/bin

# install protoc
ARG PROTOC_VERSION=3.7.0
ARG PROTOC_URL=https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-${OS}-x86_64.zip
ARG PROTOC_FILENAME=protoc.zip
RUN curl -L ${PROTOC_URL} > /tmp/${PROTOC_FILENAME}
RUN unzip -o /tmp/${PROTOC_FILENAME} -d /usr/local/
RUN rm /tmp/${PROTOC_FILENAME}

RUN go get -u github.com/golang/protobuf/protoc-gen-go && \
	go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway && \
	go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger && \
	go get -u github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc

ARG GO_VERSION=go1.12.4
RUN go get -u golang.org/dl/${GO_VERSION} && ${GO_VERSION} download