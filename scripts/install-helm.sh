#!/bin/bash

set -x

HELM_RELEASE=v3.0.2

wget https://get.helm.sh/helm-${HELM_RELEASE}-linux-amd64.tar.gz
tar zxf helm-${HELM_RELEASE}-linux-amd64.tar.gz --strip 1 linux-amd64/helm
mv helm "${HOME}"/bin/helm
rm -rf helm-${HELM_RELEASE}-linux-amd64.tar.gz
