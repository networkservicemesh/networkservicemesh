#!/usr/bin/env bash

# get scripts location
export KUBECONFIG="$( cd "$(dirname "$0")" ; pwd -P )/.kube/config"