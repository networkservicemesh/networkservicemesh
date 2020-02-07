#!/usr/bin/env bash
set -x

export KUBECONFIG=$KUBECONFIG_CLUSTER_1
make k8s-pods
make k8s-save-artifacts

# cleanup
make k8s-reset

export KUBECONFIG=$KUBECONFIG_CLUSTER_2
make k8s-pods
make k8s-save-artifacts

# cleanup
make k8s-reset
