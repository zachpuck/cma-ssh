#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

function finish
{
  "$PIPELINE_WORKSPACE"/scripts/delete-azure-infra.sh
}

# The following env vars are provided by CI.
# AZURE_CLIENT_ID
# AZURE_CLIENT_SECRET
# AZURE_TENANT_ID

# DEBUG defaults to 1
# to disable run: $ DEBUG=0 ./e2e-tests.sh
DEBUG=${DEBUG:-1}
[[ ${DEBUG} == 1 ]] && set -o xtrace

# RESET_ENV defaults to 1
# when set to 0, if the test fails, the AKS/VMs don't get removed.
# this is to allow operator to run tests/investigate after failure
# They will have to be removed manually.
RESET_ENV=${RESET_ENV:-1}
[[ ${RESET_ENV} == 1 ]] && trap finish EXIT

CMC_NAME=${CMC_NAME:-cma-ssh-$(date +%Y%m%dt%H%M%S)}

export NAME_PREFIX=${NAME_PREFIX:-$CMC_NAME}
export CLUSTER_API=${CLUSTER_API:-https://cluster-manager-api-cluster-manager-api}
export CLUSTER_NAME=${CLUSTER_NAME:-$CMC_NAME}
export K8S_VERSION=${K8S_VERSION:-1.10.6}
export AKS_K8S_VERSION=${AKS_K8S_VERSION:-1.12.6}
export CMA_CALLBACK_URL=${CMA_CALLBACK_URL:-https://example.cnct.io}
export CMA_CALLBACK_REQUESTID=${CMA_CALLBACK_REQUESTID:-12345}
export ROOT_PASSWORD=${ROOT_PASSWORD:-$(date +%c | md5sum | base64)}
export resource_group="${NAME_PREFIX}"-group
export KEY_HOME=${KEY_HOME:-$HOME}

# ssh specific inputs
export SSH_USERNAME=${SSH_USERNAME:-root}
export SSH_PASSWORD=${SSH_PASSWORD:-$ROOT_PASSWORD}
export SSH_PORT=${SSH_PORT:-22}
HELM_HOST=localhost:44134

PIPELINE_WORKSPACE=${PIPELINE_WORKSPACE:-$(cd -P -- "$(dirname -- "$0")" && cd .. && pwd -P)}

cd "$PIPELINE_WORKSPACE"

echo "login to azure"
AZURE_SUBSCRIPTION_ID=${AZURE_SUBSCRIPTION_ID:-$(az login --service-principal \
                                                   -u "${AZURE_CLIENT_ID}" \
                                                   -p "${AZURE_CLIENT_SECRET}" \
                                                   --tenant "${AZURE_TENANT_ID}" \
                                                   --output json | jq -Mr '.[].id')}
export AZURE_SUBSCRIPTION_ID

echo "set subscription"
az account set --subscription "${AZURE_SUBSCRIPTION_ID}"

# create test infra in azure
"$PIPELINE_WORKSPACE"/scripts/provision-azure-infra.sh

# shellcheck disable=SC2154
az aks get-credentials --name "$CMC_NAME" -g "$resource_group"

kubectl create clusterrolebinding superpowers \
  --clusterrole=cluster-admin \
  --user=system:serviceaccount:kube-system:default

# These need to come *after* provision-azure-infra.sh!
CONTROL_PLANE_PRIVATE_IP=${CONTROL_PLANE_PRIVATE_IP:-$(<CONTROL_PLANE_PRIVATE_IP)}
CONTROL_PLANE_PUBLIC_IP=${CONTROL_PLANE_PUBLIC_IP:-$(<CONTROL_PLANE_PUBLIC_IP)}
WORKER01_PRIVATE_IP=${WORKER01_PRIVATE_IP:-$(<WORKER01_PRIVATE_IP)}
WORKER01_PUBLIC_IP=${WORKER01_PUBLIC_IP:-$(<WORKER01_PUBLIC_IP)}
proxyIP=$(<proxyIP)
nodeIP=$(kubectl get nodes -o json | \
         jq -Mr '[.items[].status |
                 select(.conditions[].type == "Ready") |
                   .addresses[] |
                   select(.type == "InternalIP").address] | .[0]')

export HELM_HOST DEBUG CLUSTER_NAME \
       CMA_CALLBACK_REQUESTID CMA_CALLBACK_URL \
       CONTROL_PLANE_PRIVATE_IP CONTROL_PLANE_PUBLIC_IP \
       WORKER01_PRIVATE_IP WORKER01_PUBLIC_IP proxyIP

helm plugin install https://github.com/rimusz/helm-tiller || true

# for local testing
pkill tiller 2>/dev/null || true

helm tiller start-ci
helm repo add cnct https://charts.cnct.io
helm repo update

kubectl apply \
    -f https://raw.githubusercontent.com/jetstack/cert-manager/release-0.6/deploy/manifests/00-crds.yaml
kubectl create namespace cert-manager
kubectl label namespace cert-manager certmanager.k8s.io/disable-validation=true

helm --debug install --name cert-manager --namespace cert-manager stable/cert-manager --wait --timeout=600

# is this necessary?
helm --debug install --name nginx-ingress stable/nginx-ingress

helm --debug install --name cma-ssh  \
     --set images.bootstrap.tag="${PIPELINE_DOCKER_TAG}" \
     --set images.operator.tag="${PIPELINE_DOCKER_TAG}" \
     --set install.bootstrapIp="${nodeIP}" \
     --set install.airgapProxyIp="${proxyIP}" \
           cnct/cma-ssh --wait --timeout=600
helm --debug install -f "$PIPELINE_WORKSPACE"/test/e2e/manifests/cma-values.yaml --name cluster-manager-api cnct/cluster-manager-api --wait --timeout=600
helm --debug install -f "$PIPELINE_WORKSPACE"/test/e2e/manifests/cma-operator-values.yaml --name cma-operator cnct/cma-operator --wait --timeout=600

helm tiller stop

# create kubernetes job to run tests
# if envsubst is NOT found, it will be added.
if ! which envsubst >/dev/null 2>&1; then apk -U add gettext; fi

[[ $DEBUG ]] && envsubst < "$PIPELINE_WORKSPACE"/test/e2e/manifests/run-tests-job.yaml
envsubst < "$PIPELINE_WORKSPACE"/test/e2e/manifests/run-tests-job.yaml | kubectl apply -f -

# if this fails, I don't necessarily want to tear down the cluster yet.
# IE I don't want to trigger the trap.
set +e
kubectl wait --for=condition=complete job/cma-ssh-e2e-tests --timeout=36m

# output logs after job completes
kubectl logs job/cma-ssh-e2e-tests -n pipeline-tools
