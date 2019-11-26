#!/usr/bin/env bash

error(){ echo "error: $*" >&2; exit 1; }

[ -n "$*" ] && CHARTS="$*"
[ -n "$CHARTS" ] || error "chart name(s) expected as positional arguments or CHARTS variable"
[ -n "$NSM_NAMESPACE" ] || error "NSM_NAMESPACE must be set"

: "${CHECK_CMD:=make k8s-check}"

reverse() {
  { for e in "$@"; do echo "$e"; done } | tac | tr '\n' ' '
}

echo "Testing HELM chart deployment, chart(s): ${CHARTS}, namespace: ${NSM_NAMESPACE}"

for ch in $CHARTS; do
  dp="deployments/helm/$ch"
  [ -d "$dp" ] || error "Deployment $dp not found"
done

DEPLOYED_CHARTS=''

cleanup() {
  charts=$(reverse $DEPLOYED_CHARTS)
  echo "Deleting chart(s): $charts"
  for ch in $charts; do
    make "helm-delete-${ch}"
  done
  [ "$DELETE_NAMESPACES" == '1' ] && make k8s-delete-nsm-namespaces
  make k8s-deconfig k8s-config
  kubectl delete pods --force --grace-period 0 -n "${NSM_NAMESPACE}" --all
}

onfail() {
  errmsg="Helm chart deployment '$CHARTS' failed"
  echo "$errmsg"
  echo "Cleanup failed chart testing..."

  trap - ERR
  set +o errexit

  kubectl get pods -n "${NSM_NAMESPACE}"
  make k8s-logs-snapshot
  DELETE_NAMESPACES=1 cleanup

  echo "$errmsg"
  exit 1
}

trap onfail ERR
set -o errexit

make k8s-deconfig
# deploy charts
for ch in $CHARTS; do
  make "helm-install-$ch"
  DEPLOYED_CHARTS="$DEPLOYED_CHARTS $ch"
done
# run checks
if [ -n "$CHECK_CMD" ]; then
  echo "Executing checks: $CHECK_CMD"
  $CHECK_CMD
  echo "All checks are OK"
fi

trap - ERR
set +o errexit

make k8s-logs-snapshot-only-master
cleanup

echo "Helm chart(s) deployment '$CHARTS' succeeded"
