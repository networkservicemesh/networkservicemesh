#!/usr/bin/env bash

./scripts/azure/install-azure-cli.sh
az login --service-principal --username "${AZURE_SERVICE_PRINCIPAL}" --password "${AZURE_SERVICE_PRINCIPAL_SECRET}" --tenant "${CIRCLE_AZURE_TENANT}"
export AZURE_CLUSTER_NAME="nsm-ci-cluster-${CLUSTER_ID}-${CIRCLE_WORKFLOW_ID}"
export AZURE_RESOURCE_GROUP="${CIRCLE_AZURE_RESOURCE_GROUP}"
make azure-destroy

