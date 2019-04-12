#!/usr/bin/env bash

./scripts/azure/install-azure-cli.sh
az login --service-principal --username ${CIRCLE_AZURE_USERNAME} --password ${CIRCLE_AZURE_PASSWORD} --tenant ${CIRCLE_AZURE_TENANT}
export AZURE_CLUSTER_NAME="nsm-ci-cluster-${CLUSTER_ID}-${CIRCLE_WORKFLOW_ID}"
make azure-destroy

