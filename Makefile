IMAGE = quay.io/samsung_cnct/cma-ssh
VERSION = v0.1.0
KUBERNETES_VERSIONS = 1.12.6 1.13.4
DIR := ${CURDIR}
REGISTRY ?= quay.io/samsung_cnct
TARGET= cma-ssh
GOTARGET = github.com/samsung-cnct/$(TARGET)
IMAGE = $(REGISTRY)/$(TARGET)-dependencies
BUILDMNT = /go/src/$(GOTARGET)
DOCKER = docker
DOCKER_BUILD ?= $(DOCKER) run --rm -v $(DIR):$(BUILDMNT) -w $(BUILDMNT) $(IMAGE) /bin/sh -c
HOST_GOOS ?= $(shell go env GOOS)
HOST_GOARCH ?= $(shell go env GOARCH)
GO_SYSTEM_FLAGS ?= GOOS=$(HOST_GOOS) GOARCH=$(HOST_GOARCH)
KUBERNETES_LDFLAG ?= $(GOTARGET)/pkg/apis/cluster/v1alpha1.kubernetesVersions=$(KUBERNETES_VERSIONS)

all: generate test cma-ssh

.PHONY: build-dependencies-container

cma-ssh: build-dependencies-container
	$(DOCKER_BUILD) 'CGO_ENABLED=0 $(GO_SYSTEM_FLAGS) \
	go build -o $(TARGET) \
	-ldflags='\''-X "$(KUBERNETES_LDFLAG)"'\'' \
	cmd/cma-ssh/main.go'

build-dependencies-container:
	docker build -t $(IMAGE) -f build/docker/build-tools/Dockerfile build/docker/build-tools

test: build-dependencies-container
	$(DOCKER_BUILD) 'go test -v ./...'

generate: build-dependencies-container
	$(DOCKER_BUILD) 'go generate ./...'

dep-ensure: build-dependencies-container
	$(DOCKER_BUILD) 'dep ensure -v'
