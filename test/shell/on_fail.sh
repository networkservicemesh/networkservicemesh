#!/usr/bin/env bash
set -x

make k8s-pods
make k8s-save-artifacts

# cleanup
make k8s-reset
