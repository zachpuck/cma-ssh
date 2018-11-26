
# Image URL to use all building/pushing image targets
IMG ?= quay.io/samsung_cnct/cma-ssh:latest

all: test binary

# Run tests
test: generate fmt vet manifests
	go test ./pkg/... ./cmd/... -coverprofile cover.out

# Build manager binary
binary: generate fmt vet
	go build -o bin/cma-ssh github.com/samsung-cnct/cma-ssh/cmd/cma-ssh

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet
	go run ./cmd/cma-ssh/main.go

# Install CRDs into a cluster
install: manifests
	kubectl apply -f config/crds

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	kubectl apply -f config/crds
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests:
	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go all

# Run go fmt against code
fmt:
	go fmt ./pkg/... ./cmd/...

# Run go vet against code
vet:
	go vet ./pkg/... ./cmd/...

# Generate code
generate:
	go generate ./pkg/... ./cmd/...

# Build the docker image
docker-build: test
	docker build . -t ${IMG}
	@echo "updating kustomize image patch file for cma-ssh resource"
	sed -i'' -e 's@image: .*@image: '"${IMG}"'@' ./config/default/cma_ssh_image_patch.yaml

# Push the docker image
docker-push:
	docker push ${IMG}
