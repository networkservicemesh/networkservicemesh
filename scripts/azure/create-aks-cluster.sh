#!/usr/bin/env bash

readonly AZURE_RESOURCE_GROUP=$1
readonly AZURE_CLUSTER_NAME=$2
readonly AZURE_CREDENTIALS_PATH=$3
readonly AZURE_SERVICE_PRINCIPAL=$4
readonly AZURE_SERVICE_PRINCIPAL_SECRET=$5

if [[ -z "$1" ]] || [[ -z "$2" ]] || [[ -z "$3" ]]; then
    echo "Usage: create-aks-cluster.sh <resource-group> <cluster-name> <kube-config-path> [<service-principal> <password>]"
    exit 1
fi

echo -n "Creating AKS cluster '$AZURE_CLUSTER_NAME'..."

if (az aks show --resource-group "$AZURE_RESOURCE_GROUP" --name "$AZURE_CLUSTER_NAME" > /dev/null 2>&1); then
    echo "already exists"
else
    echo
    if [[ -z "$AZURE_SERVICE_PRINCIPAL" ]] || [[ -z "$AZURE_SERVICE_PRINCIPAL_SECRET" ]]; then
        az aks create \
            --resource-group "$AZURE_RESOURCE_GROUP" \
            --name "$AZURE_CLUSTER_NAME" \
            --node-count 2 \
            --node-vm-size Standard_B2s \
            --generate-ssh-keys \
            --enable-rbac \
            --no-wait && \
        echo "done" || exit 1
    else
	     az aks create \
            --resource-group "$AZURE_RESOURCE_GROUP" \
            --name "$AZURE_CLUSTER_NAME" \
            --node-count 2 \
            --node-vm-size Standard_B2s \
            --generate-ssh-keys \
            --enable-rbac \
            --no-wait \
            --service-principal "$AZURE_SERVICE_PRINCIPAL" \
            --client-secret "$AZURE_SERVICE_PRINCIPAL_SECRET" && \
        echo "done" || exit 1
    fi
fi

echo "Waiting for deploy to complete..."
az aks wait  \
   	--name "$AZURE_CLUSTER_NAME" \
   	--resource-group "$AZURE_RESOURCE_GROUP" \
	--created > /dev/null && \
echo "done" || exit 1

mkdir -p "$(dirname "$AZURE_CREDENTIALS_PATH")"
az aks get-credentials \
   	--name "$AZURE_CLUSTER_NAME" \
   	--resource-group "$AZURE_RESOURCE_GROUP" \
   	--file "$AZURE_CREDENTIALS_PATH" \
   	--overwrite-existing || exit 1

