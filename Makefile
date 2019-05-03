IMAGE = quay.io/samsung_cnct/cma-ssh
VERSION = v0.1.0
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
GO = go1.12.4
GO_SYSTEM_FLAGS ?= GOOS=$(HOST_GOOS) GOARCH=$(HOST_GOARCH) GO111MODULE=on
GOFILES = $(shell find ./ -type f -name '*.go')

all: cma-ssh

clean:
	rm -rf bin
	rm -f cma-ssh

$(GO):
	GO111MODULE=off go get -u golang.org/dl/$(GO)
	GO111MODULE=off $(GO) download

cma-ssh: $(GOFILES)
	CGO_ENABLED=0 $(GO_SYSTEM_FLAGS) $(GO) build -o $(TARGET) cmd/cma-ssh/main.go

bin:
	mkdir bin

bin/deepcopy-gen: bin
	$(GO_SYSTEM_FLAGS) GOBIN=$(DIR)/bin $(GO) install k8s.io/code-generator/cmd/deepcopy-gen

.PHONY: build-dependencies-container

build-dependencies-container:
	docker build -t $(IMAGE) -f build/docker/build-tools/Dockerfile build/docker/build-tools

test: build-dependencies-container
	$(DOCKER_BUILD) 'go test -v ./...'

generate: bin/deepcopy-gen
	PATH=${CURDIR}/bin:$(PATH) $(GO) generate ./...

clean-test: build-dependencies-container
	$(DOCKER_BUILD) 'make go1.12.4 && make'
