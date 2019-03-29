#!/usr/bin/env bash

export AZURE_RESOURCE_GROUP=$1
export AZURE_CLUSTER_NAME=$2
export AZURE_CREDENTIALS_PATH=$3

echo -n "Creating AKS cluster $AZURE_CLUSTER_NAME..."

if (az aks show --resource-group "$AZURE_RESOURCE_GROUP" --name "$AZURE_CLUSTER_NAME" > /dev/null 2>&1); then
	echo "already exists"
else
	echo
	az aks create \
		--resource-group "$AZURE_RESOURCE_GROUP" \
		--name "$AZURE_CLUSTER_NAME" \
		--node-count 2 \
		--node-vm-size Standard_B2s \
		--enable-addons monitoring \
		--generate-ssh-keys && \
	echo "done" || exit 1
fi

echo "Waiting for deploy to complete..."
az aks wait  \
   	--name "$AZURE_CLUSTER_NAME" \
   	--resource-group "$AZURE_RESOURCE_GROUP" \
	--created > /dev/null && \
echo "done" || exit 1

mkdir -p "$AZURE_CREDENTIALS_PATH"
az aks get-credentials \
   	--name "$AZURE_CLUSTER_NAME" \
   	--resource-group "$AZURE_RESOURCE_GROUP" \
   	--file "$AZURE_CREDENTIALS_PATH/config" \
   	--overwrite-existing || exit 1

