#!/usr/bin/env bash

# configure CLUSTER 1
# helm tests expect cluster to be clean
export KUBECONFIG=$KUBECONFIG_CLUSTER_1

make k8s-deconfig

make helm-install-nsm || exit $?
make helm-install-proxy-nsmgr || exit $?
make helm-install-endpoint || exit $?

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

make helm-install-nsm || exit $?
make helm-install-proxy-nsmgr || exit $?
export NETWORK_SERVICE="icmp-responder@$CLUSTER1_NODE_IP"
make helm-install-client || exit $?

# check connection
make k8s-icmp-check || exit $?

export KUBECONFIG=$KUBECONFIG_CLUSTER_1
# collect logs for correct test execution
make k8s-logs-snapshot-only-master

# cleanup
make helm-delete k8s-terminating-cleanup

# restore CRDs and RBAC
make k8s-deconfig k8s-config

export KUBECONFIG=$KUBECONFIG_CLUSTER_2
# collect logs for correct test execution
make k8s-logs-snapshot-only-master

# cleanup
make helm-delete k8s-terminating-cleanup

# restore CRDs and RBAC
make k8s-deconfig k8s-config
