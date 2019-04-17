# Azure Quick Start

## Running NSM integration tests on Azure

All the steps below could be done via Azure [portal](https://portal.azure.com). 
This document mostly shows how to run NSM tests using the Azure's CLI tool. 

#### Get Azure Account

Make sure you have an Azure account with enough privileges to create resource groups and 
[AKS](https://docs.microsoft.com/en-us/azure/aks/) clusters. You may register one for free. 

#### Install CLI Tool

Follow the [instructions](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli?view=azure-cli-latest) 
to install Azure's `az` CLI tool

If you're using Debian or Ubunty you'll probably want to simply run `./scripts/azure/install-azure-cli.sh`

#### Sign in with CLI Tool

Follow the [instructions](https://docs.microsoft.com/en-us/cli/azure/authenticate-azure-cli) to sign in with the Azure CLI

#### Define a Resource Group

Create a new resource group (if you don't have one)
```bash
az group create --location centralus --name nsm-resource-group
``` 
Example Output:
```json
{
  "id": "/subscriptions/eb8583f9-56c6-4b83-9903-ac8be7c1a9de/resourceGroups/nsm-resource-group",
  "location": "centralus",
  "managedBy": null,
  "name": "nsm-resource-group",
  "properties": {
    "provisioningState": "Succeeded"
  },
  "tags": null,
  "type": null
}
```
Run `az account list-locations` to list all available locations. 

Run `az group list` or `az group list -o table` to list all existing groups.

See the corresponding [docs](https://docs.microsoft.com/en-us/cli/azure/group?view=azure-cli-latest#az-group-create) for more details.

#### Start and Configure AKS Cluster

To create and configure AKS cluster run the following commands:
```bash
source .env/azure.env
source scripts/azure/env.sh
make k8s-config
```
This will create an AKS cluster with 2 nodes (2 Cores, 8GB RAM each), 
apply required kubernetes config and save credentials in `scripts/azure/.kube/config`.

To simply create AKS cluster simply run:
```bash
make azure-start
```

Environment variables that affects `azure-start` goal:
1. `AZURE_RESOURCE_GROUP` - azure resource group to use (default is `nsm-ci`)
2. `AZURE_CLUSTER_NAME` - name of AKS cluster to be create (default is `nsm-ci-cluster`)
3. `AZURE_CREDENTIALS_PATH` - a path to store kubernetes credentials (default is `scripts/azure/.kube/config`)
4. `AZURE_SERVICE_PRINCIPAL` - an id of service-principal to create cluster (optional, not set by default)
5. `AZURE_SERVICE_PRINCIPAL_SECRET` - a service-principal password (required if `AZURE_SERVICE_PRINCIPAL`)

#### Stop AKS Cluster

Simply run 
```bash
make azure-destroy
```

Environment variables that affects `azure-destroy` goal:
1. `AZURE_RESOURCE_GROUP` - azure resource group in which cluster is defined (default is `nsm-resource-group`)
2. `AZURE_CLUSTER_NAME` - AKS cluster to destroy (default is `nsm-cluster`)

## Using Service Principal

Service principals are accounts not tied to any particular user. SPs have it's own permissions and roles 
(with respect to scoped Azure resources) and this is a recommended way to access Azure from automatic services (e.g. from CI)

#### Create a Service principal

Execute `ad sp create-for-rbac --name <principal-name>`. E.g.:
```bash
az ad sp create-for-rbac --name nsm-ci-service-principal
```
Example Output:
```json
{
  "appId": "1fe55163-6f8c-4592-8e9f-5b9cab7e39f4",
  "displayName": "nsm-ci-service-principal",
  "name": "http://nsm-ci-service-principal",
  "password": "f0d6d3ce-b72e-430a-972f-025b2cc7279e",
  "tenant": "5a60fd29-7786-4a74-a1d6-9c9d894b1881"
}
```
**NB**: This credentials must be saved. It **cannot** retrieved later.

See [documentation](https://docs.microsoft.com/en-us/cli/azure/create-an-azure-service-principal-azure-cli?view=azure-cli-latest)
for more details.

#### Grant Privileges

Service-principal needs to be an owner of the resource-group in which AKS clusters will be created.

First get resource-group id. Run `az group show --name <group-name>`, e.g.:
```bash
az group show --name nsm-ci
```
Example Output:
```json
{
  "id": "/subscriptions/eb8583f9-56c6-4b83-9903-ac8be7c1a9de/resourceGroups/nsm-ci",
  "location": "centralus",
  "managedBy": null,
  "name": "nsm-ci",
  "properties": {
    "provisioningState": "Succeeded"
  },
  "tags": null,
  "type": null
}
```

Next, assign service-principal an `Owner` role of the resource-group you want. 

Run `az role assignment create --assignee <principal-app-id> --scope <group-id> --role Owner`, e.g.:
```bash
az role assignment create \ 
    --assignee 1fe55163-6f8c-4592-8e9f-5b9cab7e39f4 \
    --scope /subscriptions/eb8583f9-56c6-4b83-9903-ac8be7c1a9de/resourceGroups/nsm-ci \
    --role Owner
```
Example Output:
```json
{
  "canDelegate": null,
  "id": "/subscriptions/eb8583f9-56c6-4b83-9903-ac8be7c1a9de/resourceGroups/nsm-ci/providers/Microsoft.Authorization/roleAssignments/4415ee4f-9b81-4a54-9b1b-0cf2eabde10c",
  "name": "4415ee4f-9b81-4a54-9b1b-0cf2eabde10c",
  "principalId": "3247f891-1406-44ea-8870-331fa0bf524f",
  "resourceGroup": "nsm-ci",
  "roleDefinitionId": "/subscriptions/eb8583f9-56c6-4b83-9903-ac8be7c1a9de/providers/Microsoft.Authorization/roleDefinitions/8e3af657-a8ff-443c-a75c-2fe8c4bcb635",
  "scope": "/subscriptions/eb8583f9-56c6-4b83-9903-ac8be7c1a9de/resourceGroups/nsm-ci",
  "type": "Microsoft.Authorization/roleAssignments"
}
```
See [documentation](https://docs.microsoft.com/en-us/cli/azure/role/assignment?view=azure-cli-latest#az-role-assignment-create) 
for more details.

#### Login Azure CLI with Service Principal

Run `az login --service-principal -u <app-url> -p <password> --tenant <tenant>`, e.g.:
```bash
az login \
    --service-principal \
    -u 1fe55163-6f8c-4592-8e9f-5b9cab7e39f4 \
    -p f0d6d3ce-b72e-430a-972f-025b2cc7279e \
    --tenant 5a60fd29-7786-4a74-a1d6-9c9d894b1881
```

#### Create AKS Cluster using Service Principal

Run `az aks create --resource-group <group> --name <cluster-name> --service-principal <app-id> --client-secret <password>`, e.g.:

```bash
az aks create \
    --resource-group nsm-ci \
    --name nsm-ci-cluster \
    --service-principal 1fe55163-6f8c-4592-8e9f-5b9cab7e39f4 \
    --client-secret f0d6d3ce-b72e-430a-972f-025b2cc7279e
```
See [documentation](https://docs.microsoft.com/en-us/azure/aks/kubernetes-service-principal#specify-a-service-principal-for-an-aks-cluster)
for more details.

Using `make` machinery:
```bash
export AZURE_RESOURCE_GROUP=nsm-ci
export AZURE_CLUSTER_NAME=nsm-ci-cluster
export AZURE_SERVICE_PRINCIPAL=1fe55163-6f8c-4592-8e9f-5b9cab7e39f4
export AZURE_SERVICE_PRINCIPAL_SECRET=f0d6d3ce-b72e-430a-972f-025b2cc7279e
make azure-start
```
