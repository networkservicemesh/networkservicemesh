#!/bin/bash

set -x

function prepare() {
  echo "Preparing cluster..."

  make k8s-config
  make helm-init
  make spire-install
}

# workaround for https://github.com/helm/helm/issues/6361:
function delete_unavailable_apiservice() {
  for service in $(kubectl get apiservice | grep False | awk '{print $1}'); do
    echo "Deleting ${service}..."
    kubectl delete apiservice "${service}"
  done
}

if ! prepare; then
  delete_unavailable_apiservice
  prepare
fi
