#!/usr/bin/env bash

# helm tests expect cluster to be clean
make k8s-deconfig

make helm-install-nsm || exit $?
make helm-install-icmp-responder || exit $?
make helm-install-crossconnect-monitor || exit $?

make k8s-crossconnect-monitor-check || exit $?

# collect logs for correct test execution
make k8s-logs-snapshot-only-master

# cleanup
make helm-delete-crossconnect-monitor
make helm-delete-icmp-responder
make helm-delete-nsm
kubectl delete pods --force --grace-period 0 -n "${NSM_NAMESPACE}" --all

# restore CRDs and RBAC
make k8s-deconfig k8s-config
