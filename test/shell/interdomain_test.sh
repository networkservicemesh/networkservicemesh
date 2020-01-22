#!/usr/bin/env bash

# configure CLUSTER 1
# helm tests expect cluster to be clean
export KUBECONFIG=$KUBECONFIG_CLUSTER_1

make k8s-deconfig
read -r -a HELM_TEST_OPTS < <(make helm-test-opts)

helm install nsm deployments/helm/nsm "${HELM_TEST_OPTS[@]}" || exit $?
helm install proxy-nsmgr deployments/helm/proxy-nsmgr "${HELM_TEST_OPTS[@]}" || exit $?
helm install endpoint deployments/helm/endpoint "${HELM_TEST_OPTS[@]}" || exit $?

# get external ip address of CLUSTER 1 node
# shellcheck disable=SC2089 disable=SC1083 disable=SC2102
CLUSTER1_NODE_IP=$(kubectl get nodes -o jsonpath={.items[0].status.addresses[?\(@.type==\"ExternalIP\"\)].address})
if [ -z "$CLUSTER1_NODE_IP" ]; then
    # shellcheck disable=SC2089 disable=SC1083 disable=SC2102
    CLUSTER1_NODE_IP=$(kubectl get nodes -o jsonpath={.items[0].status.addresses[?\(@.type==\"InternalIP\"\)].address})
fi

# configure CLUSTER 2
export KUBECONFIG=$KUBECONFIG_CLUSTER_2

make k8s-deconfig

helm install nsm deployments/helm/nsm "${HELM_TEST_OPTS[@]}" || exit $?
helm install proxy-nsmgr deployments/helm/proxy-nsmgr "${HELM_TEST_OPTS[@]}" || exit $?
helm install client deployments/helm/client "${HELM_TEST_OPTS[@]}" \
             --set networkservice="icmp-responder@$CLUSTER1_NODE_IP" || exit $?

# check connection
make k8s-icmp-check || exit $?

export KUBECONFIG=$KUBECONFIG_CLUSTER_1
# collect logs for correct test execution
make k8s-save-artifacts-only-master

# cleanup
make k8s-reset

export KUBECONFIG=$KUBECONFIG_CLUSTER_2
# collect logs for correct test execution
make k8s-save-artifacts-only-master

# cleanup
make k8s-reset
