#!/usr/bin/env bash

# helm tests expect cluster to be clean
make k8s-deconfig

make helm-install-nsm || exit $?
make helm-install-vpn || exit $?
make k8s-vpn-check || exit $?

# collect logs for correct test execution
make k8s-save-artifacts-only-master

# cleanup
make helm-delete k8s-terminating-cleanup

# restore CRDs and RBAC
make k8s-deconfig k8s-config
