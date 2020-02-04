#!/usr/bin/env bash
set -x
# helm tests expect cluster to be clean
make k8s-deconfig
read -r -a HELM_TEST_OPTS < <(make helm-test-opts)

helm install deployments/helm/nsm "${HELM_TEST_OPTS[@]}" || exit $?
helm install deployments/helm/endpoint "${HELM_TEST_OPTS[@]}" || exit $?
helm install deployments/helm/client "${HELM_TEST_OPTS[@]}" || exit $?

make k8s-icmp-check || exit $?

# collect logs for correct test execution
make k8s-save-artifacts-only-master

# cleanup
make helm-delete k8s-terminating-cleanup

# restore CRDs and RBAC
make k8s-deconfig k8s-config
