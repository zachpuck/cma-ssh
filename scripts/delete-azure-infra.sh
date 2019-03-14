#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail
[[ ${DEBUG:-1} ]] && set -o xtrace

echo "logging to azure"
az login --service-principal -u "${AZURE_CLIENT_ID}" -p "${AZURE_CLIENT_SECRET}" --tenant "${AZURE_TENANT_ID}"

echo "setting subscription"
az account set --subscription "${AZURE_SUBSCRIPTION_ID}"

echo "deleting resource group"
az group delete --name "${NAME_PREFIX}"-group --yes --no-wait
