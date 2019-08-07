#!/bin/bash

set -x

HELM_RELEASE=v2.14.3

wget -c https://storage.googleapis.com/kubernetes-helm/helm-${HELM_RELEASE}-linux-amd64.tar.gz
tar zxf helm-${HELM_RELEASE}-linux-amd64.tar.gz --strip 1 linux-amd64/helm
mv helm "${HOME}"/bin/helm
