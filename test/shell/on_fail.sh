#!/usr/bin/env bash

make k8s-pods
make k8s-logs-snapshot

# cleanup
make helm-delete k8s-terminating-cleanup
make k8s-delete-nsm-namespaces

# restore CRDs and RBAC
make k8s-deconfig k8s-config
