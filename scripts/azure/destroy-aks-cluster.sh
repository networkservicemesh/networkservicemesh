#!/usr/bin/env bash

export AZURE_RESOURCE_GROUP=$1
export AZURE_CLUSTER_NAME=$2

echo -n "Destroying AKS cluster $AZURE_CLUSTER_NAME..."

if ! (az aks show --resource-group "$AZURE_RESOURCE_GROUP" --name "$AZURE_CLUSTER_NAME" > /dev/null 2>&1); then
	echo "not found"
else
	limit=10;
	attempt=1;

	until test "$attempt" -gt "$limit"  || 	az aks delete \
		--name "$AZURE_CLUSTER_NAME" \
		--resource-group "$AZURE_RESOURCE_GROUP" \
		--yes --no-wait; do
		attempt=$(( attempt + 1 ));
    	rm -rf "$GOPATH"/pkg/mod/cache/vcs/* # wipe out the vcs cache to overwrite corrupted repos
    	test "$attempt" -le "$limit" && echo "Trying again, attempt $attempt";
	done
	test "$attempt" -le "$limit" # ensure correct exit code
	echo "done"
fi