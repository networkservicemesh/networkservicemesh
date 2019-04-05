#!/usr/bin/env bash

# get scripts location
KUBECONFIG="$( cd "$(dirname "$0")" && pwd -P )/.kube/config"
export KUBECONFIG