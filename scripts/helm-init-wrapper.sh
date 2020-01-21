#!/bin/bash

# A script to insulate users from differences between Helm 2 and 3. In Helm 3
# there is no init and no tiller, so when using Helm 3 we simply skip these
# steps.

HELM_VERSION=$(helm version | awk -v FS="(Ver\"|\")" '{print$ 2}')


if [[ $HELM_VERSION = v2* ]]
then
  helm init --wait && ./scripts/helm-patch-tiller.sh
elif [[ $HELM_VERSION = v3* ]]
then
  echo "Using Helm 3, skipping 'helm init'..."
else
  echo "Unsupported helm version: $HELM_VERSION"
fi

