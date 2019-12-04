#!/bin/bash

if [ -z "${CLUSTER_RULES_PREFIX}" ]; then
  export CLUSTER_RULES_PREFIX="vagrant"
fi
if [ -n "$ZSH_VERSION" ]; then
  RESOLVED_SRC="${(%):-%N}"
else
  RESOLVED_SRC="${BASH_SOURCE[0]}"
fi

DIR="$( cd "$( dirname "${RESOLVED_SRC}" )" >/dev/null && pwd )"
export KUBECONFIG=${DIR}/.kube/config
