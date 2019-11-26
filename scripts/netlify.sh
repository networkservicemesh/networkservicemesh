#!/bin/bash

export PATH="${PATH}:./helm-output/bin/"
mkdir -p ./helm-output/bin
# Install helm
echo "Installing helm"
curl -L https://git.io/get_helm.sh | USE_SUDO="false" HELM_INSTALL_DIR="./helm-output/bin" bash
helm init --client-only
git tag netlify
git remote add upstream https://github.com/networkservicemesh/networkservicemesh.git
git fetch upstream
TAGS=(master v0.1.0 v0.2.0)
for TAG in "${TAGS[@]}"
do
   git checkout "${TAG}" || continue
   mkdir -p ./helm-output/"${TAG}"
   helm repo index ./helm-output/"${TAG}"
   helm package -d ./helm-output/"${TAG}" ./deployments/helm/*
   helm repo index ./helm-output/"${TAG}"
done
rm -r ./helm-output/bin
git checkout netlify
