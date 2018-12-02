GOCMD=go
DOCKERTAG?=quay.io/samsung_cnct/cma-ssh:local
BOOTSTRAPDOCKERTAG?=quay.io/samsung_cnct/cma-ssh-bootstrap:local
GOBUILD=$(GOCMD) build
GOGET=$(GOCMD) get
GOTEST=$(GOCMD) test
PROJECTDIR=$(shell pwd)
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
	output=$$(kustomize build build/kustomize/crd/protected/cluster); echo "$$output" > $(PROJECTDIR)/deployments/helm/cma-ssh/CRD-protected/cluster_v1alpha1_cluster.yaml
	output=$$(kustomize build build/kustomize/crd/protected/machine); echo "$$output" > $(PROJECTDIR)/deployments/helm/cma-ssh/CRD-protected/custer_v1alpha1_machine.yaml
	output=$$(kustomize build build/kustomize/crd/unprotected/cluster); echo "$$output" > $(PROJECTDIR)/deployments/helm/cma-ssh/CRD/cluster_v1alpha1_cluster.yaml
	output=$$(kustomize build build/kustomize/crd/unprotected/machine); echo "$$output" > $(PROJECTDIR)/deployments/helm/cma-ssh/CRD/cluster_v1alpha1_machine.yaml

# Run go fmt against code
fmt:
	go fmt ./pkg/... ./cmd/...

# Run go vet against code
vet:
	go vet ./pkg/... ./cmd/...

# Generate code
generate:
	go generate ./pkg/... ./cmd/...

# Install dep will install dependencies
install-dep:
	go get -u github.com/golang/dep/cmd/dep
	go get -u github.com/onsi/ginkgo/ginkgo
	go get -u github.com/onsi/gomega/...

# New Dep will just blow away Gopkg, vendor and do a dep init
new-dep:
	rm -rf vendor Gopkg.toml Gopkg.lock
	$(DEPCMD) init

# For builds where we're just going to go along with whatever Gopkg is saying
dep-ensure:
	$(DEPCMD) ensure

# Build the docker image
docker-build:
	docker build $(PROJECTDIR) -f $(PROJECTDIR)/build/docker/cma-ssh/Dockerfile -t $(DOCKERTAG)
	docker build $(PROJECTDIR) -f $(PROJECTDIR)/build/docker/bootstrap/Dockerfile -t $(BOOTSTRAPDOCKERTAG)

# Push the docker image
docker-push:
	docker push ${DOCKERTAG}
	docker push ${BOOTSTRAPDOCKERTAG}