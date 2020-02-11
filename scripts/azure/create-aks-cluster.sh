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

function register_feature {
    echo "Registering feature $1 in namespace $2..."
    REQUEST="az feature register --namespace $2 -n $1 --query properties.state -o tsv"
    while [ "$($REQUEST)" == "Registering" ]; do
        sleep 5
    done
    echo "Registered"
}

# Prepare azure for Public IP enabling
az extension add --name aks-preview
register_feature VMSSPreview Microsoft.ContainerService
register_feature NodePublicIPPreview Microsoft.ContainerService
az provider register -n Microsoft.ContainerService

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
            --debug \
            --no-wait && \
        echo "done" || exit 1
    else
        # Temporary while az aks create does not support api-version=2019-06-01 required for enableNodePublicIP option
	    az rest -m put -u "https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/$AZURE_RESOURCE_GROUP/providers/Microsoft.ContainerService/managedClusters/$AZURE_CLUSTER_NAME?api-version=2019-06-01" \
            --headers Content-Type="application/json" Accept="application/json" accept-language="en-US" -b "
            {
                \"location\": \"centralus\",
                \"properties\":
                {
                    \"kubernetesVersion\": \"1.16.4\",
                    \"dnsPrefix\": \"${AZURE_CLUSTER_NAME::10}-${AZURE_RESOURCE_GROUP}\",
                    \"agentPoolProfiles\":
                    [{
                        \"name\": \"nodepool1\",
                        \"count\": 2,
                        \"vmSize\": \"Standard_B2s\",
                        \"osType\": \"Linux\",
                        \"enableNodePublicIP\": true,
                        \"type\": \"VirtualMachineScaleSets\"
                    }],
                    \"servicePrincipalProfile\":
                    {
                        \"adminUsername\": \"azureuser\",
                        \"osType\": \"Linux\",
                        \"clientId\": \"$AZURE_SERVICE_PRINCIPAL\",
                        \"secret\": \"$AZURE_SERVICE_PRINCIPAL_SECRET\"
                    },
                    \"addonProfiles\": {},
                    \"enableRBAC\": true
                }
            }" && \
        echo "done" || exit 1
    fi
fi

echo "Waiting for deploy to complete..."
az aks wait  \
   	--name "$AZURE_CLUSTER_NAME" \
   	--resource-group "$AZURE_RESOURCE_GROUP" \
	--created > /dev/null && \
echo "done" || exit 1

echo "Creating Inbound traffic rule"
NODE_RESOURCE_GROUP=$(az aks show -g "$AZURE_RESOURCE_GROUP" -n "$AZURE_CLUSTER_NAME" --query nodeResourceGroup -o tsv)
NSG_NAME=$(az network nsg list -g "$NODE_RESOURCE_GROUP" --query "[].name" -o tsv)
az network nsg rule create --name "${NSG_NAME}-rule" \
    --nsg-name "$NSG_NAME" \
    --priority 100 \
    --resource-group "$NODE_RESOURCE_GROUP" \
    --access Allow \
    --description "Allow All Inbound Internet traffic" \
    --destination-address-prefixes '*' \
    --destination-port-ranges '*' \
    --direction Inbound \
    --protocol '*' \
    --source-address-prefixes Internet \
    --source-port-ranges '*' && \
echo "done" || exit 1

mkdir -p "$(dirname "$AZURE_CREDENTIALS_PATH")"
az aks get-credentials \
   	--name "$AZURE_CLUSTER_NAME" \
   	--resource-group "$AZURE_RESOURCE_GROUP" \
   	--file "$AZURE_CREDENTIALS_PATH" \
   	--overwrite-existing || exit 1
