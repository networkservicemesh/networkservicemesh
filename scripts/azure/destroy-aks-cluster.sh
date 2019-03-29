#!/usr/bin/env bash

export AZURE_RESOURCE_GROUP=$1
export AZURE_CLUSTER_NAME=$2

echo -n "Destroying AKS cluster $AZURE_CLUSTER_NAME..."

if ! (az aks show --resource-group "$AZURE_RESOURCE_GROUP" --name "$AZURE_CLUSTER_NAME" > /dev/null 2>&1); then
	echo "not found"
else
	echo
	az aks delete \
		--name "$AZURE_CLUSTER_NAME" \
		--resource-group "$AZURE_RESOURCE_GROUP" \
		--yes && \
	echo "done" || exit 1
fi