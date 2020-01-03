#!/usr/bin/env bash

# helm tests expect cluster to be clean
make k8s-deconfig

make helm-install-nsm || exit $?
make helm-install-icmp-responder || exit $?
make k8s-icmp-check || exit $?

# collect logs for correct test execution
make k8s-logs-snapshot-only-master

# cleanup
make helm-delete k8s-terminating-cleanup

# restore CRDs and RBAC
make k8s-deconfig k8s-config
