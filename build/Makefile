GOCMD=go
GOBUILD=$(GOCMD) build
GOGET=$(GOCMD) get
GOTEST=$(GOCMD) test
PROJECTDIR=$(shell pwd)
VERSIONFILE=$(shell cat ${PROJECTDIR}/.versionfile)
DOCKERTAG?=quay.io/samsung_cnct/cma-ssh:$(VERSIONFILE)-local
BOOTSTRAPDOCKERTAG?=quay.io/samsung_cnct/cma-ssh-bootstrap:$(VERSIONFILE)-local
DEPCMD=dep

# Variables for binary version information
GIT_COMMIT=$(shell git rev-parse HEAD)
APP_VERSION=$(shell cat .versionfile)
BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# Associating version information variables with data
VERSION_PACKAGE=github.com/samsung-cnct/cma-ssh/pkg/version
LD_DATA_GIT_COMMIT=-X ${VERSION_PACKAGE}.gitCommit=${GIT_COMMIT}
LD_DATA_VERSION=-X ${VERSION_PACKAGE}.version=${APP_VERSION}
LD_DATA_BUILD_DATE=-X ${VERSION_PACKAGE}.buildDate=${BUILD_DATE}
LD_DATA=${LD_DATA_GIT_COMMIT} ${LD_DATA_VERSION} ${LD_DATA_BUILD_DATE}

# Creating static binary
STATIC_FLAGS=-w -extldflags "-static"

# Generating ldflag string
LDFLAGS='${STATIC_FLAGS} ${LD_DATA}'

# Location of
OPERATOR_BINARY_NAME=cma-ssh
OPERATOR_SOURCE_FILE=cmd/cma-ssh/main.go

# build binary command
BUILD_OPERATOR_BINARY_CMD=${GOBUILD} -a -tags netgo -ldflags ${LDFLAGS} -o ${OPERATOR_BINARY_NAME} ${OPERATOR_SOURCE_FILE}

# protoc variables
PROTOC_VERSION=3.6.1
PROTOC_DARWIN_FILENAME=protoc-${PROTOC_VERSION}-osx-x86_64.zip
PROTOC_DARWIN_DOWNLOAD_URL=https://github.com/google/protobuf/releases/download/v${PROTOC_VERSION}/${PROTOC_DARWIN_FILENAME}

all: test cma-ssh-linux-amd64
linux: test cma-ssh-linux-amd64
darwin: test cma-ssh-darwin-amd64
container: docker-build

# Run tests
test: manifests generate fmt vet
	ginkgo -r ./pkg/... ./cmd/... -coverprofile cover.out

# Build manager binary
cma-ssh-linux-amd64: manifests generate fmt vet
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 ${BUILD_OPERATOR_BINARY_CMD}
cma-ssh-darwin-amd64: manifests generate fmt vet
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 ${BUILD_OPERATOR_BINARY_CMD}

# Run against the configured Kubernetes cluster in ~/.kube/config
run: manifests generate fmt vet
	go run ./cmd/manager/main.go

# Generate manifests e.g. CRD, RBAC etc.
# generate parts of helm chart
manifests:
	go run $(PROJECTDIR)/vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go crd --output-dir $(PROJECTDIR)/crd
	go run $(PROJECTDIR)/vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go rbac --name rbac --output-dir $(PROJECTDIR)/rbac
	output=$$(kustomize build build/kustomize/rbac/role); echo "$$output" > $(PROJECTDIR)/deployments/helm/cma-ssh/RBAC/rbac_role.yaml
	output=$$(kustomize build build/kustomize/rbac/rolebinding); echo "$$output" > $(PROJECTDIR)/deployments/helm/cma-ssh/RBAC/rbac_role_binding.yaml
	output=$$(kustomize build build/kustomize/crd/protected/cluster); echo "$$output" > $(PROJECTDIR)/deployments/helm/cma-ssh/CRD-protected/cluster_v1alpha1_cnctcluster.yaml
	output=$$(kustomize build build/kustomize/crd/protected/machine); echo "$$output" > $(PROJECTDIR)/deployments/helm/cma-ssh/CRD-protected/custer_v1alpha1_cnctmachine.yaml
	output=$$(kustomize build build/kustomize/crd/unprotected/cluster); echo "$$output" > $(PROJECTDIR)/deployments/helm/cma-ssh/CRD/cluster_v1alpha1_cnctcluster.yaml
	output=$$(kustomize build build/kustomize/crd/unprotected/machine); echo "$$output" > $(PROJECTDIR)/deployments/helm/cma-ssh/CRD/cluster_v1alpha1_cnctmachine.yaml

# Run go fmt against code
fmt:
	go fmt ./pkg/... ./cmd/...

# Run go vet against code
vet:
	go vet ./pkg/... ./cmd/...

install-tools-darwin: install-protoc-darwin install-dep

install-protoc-darwin:
	curl -L ${PROTOC_DARWIN_DOWNLOAD_URL} > /tmp/${PROTOC_DARWIN_FILENAME}
	sudo unzip -o /tmp/${PROTOC_DARWIN_FILENAME} -d /usr/local/
	rm /tmp/${PROTOC_DARWIN_FILENAME}

# Generate code
generate:
	./build/generators/api.sh
	./build/generators/swagger-dist-adjustment.sh
	go generate ./pkg/... ./cmd/...

# Install dep will install dependencies
install-dep:
	go get -u github.com/golang/dep/cmd/dep
	dep ensure
	go install ./vendor/github.com/onsi/ginkgo/ginkgo
	go install ./vendor/github.com/onsi/gomega/...
	go install ./vendor/github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc
	go install ./vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
	go install ./vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
	go install ./vendor/github.com/golang/protobuf/protoc-gen-go

# New Dep will just blow away Gopkg, vendor and do a dep init
new-dep:
	rm -rf vendor Gopkg.toml Gopkg.lock
	$(DEPCMD) init

# For builds where we're just going to go along with whatever Gopkg is saying
dep-ensure:
	$(DEPCMD) ensure

docker-build-operator:
	docker build $(PROJECTDIR) -f $(PROJECTDIR)/build/docker/cma-ssh/Dockerfile -t $(DOCKERTAG)

docker-build-bootstrap:
	docker build $(PROJECTDIR)/build/docker/bootstrap -f $(PROJECTDIR)/build/docker/bootstrap/Dockerfile -t $(BOOTSTRAPDOCKERTAG)

# Build the docker image
docker-build: docker-build-operator docker-build-bootstrap

docker-push-operator:
	docker push ${DOCKERTAG}

docker-push-bootstrap:
	docker push ${BOOTSTRAPDOCKERTAG}

# Push the docker image
docker-push: docker-push-bootstrap docker-push-operator
	@echo Install chart with local tags:
	@echo helm install deployments/helm/cma-ssh --name cma-ssh \\
	@echo --set images.operator.tag=$(VERSIONFILE)-local \\
	@echo --set images.bootstrap.tag=$(VERSIONFILE)-local