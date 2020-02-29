#!/usr/bin/env bash

# helm tests expect cluster to be clean
make k8s-deconfig
read -r -a HELM_TEST_OPTS < <(make helm-test-opts)

helm install deployments/helm/nsm "${HELM_TEST_OPTS[@]}" || exit $?
helm install deployments/helm/endpoint "${HELM_TEST_OPTS[@]}" || exit $?
helm install deployments/helm/client "${HELM_TEST_OPTS[@]}" || exit $?
helm install deployments/helm/crossconnect-monitor "${HELM_TEST_OPTS[@]}" || exit $?

make k8s-crossconnect-monitor-check || exit $?

# collect logs for correct test execution
make k8s-save-artifacts-only-master

# cleanup
make k8s-reset
