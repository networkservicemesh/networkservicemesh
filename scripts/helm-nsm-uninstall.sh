#!/bin/bash

# A script for wrapping NSM clenaup via Helm. Both Helm 2 and Helm 3 are
# supported.

function usage () {
  echo "Usage: $0 --nsm_namespace <namespace> [--char <chart name> | --all | --help]"
}

function check_flags () {
  SHOW_USAGE=0
  if [ -z ${NSM_NAMESPACE+x} ]; then
    echo "--nsm_namespace is required."
    SHOW_USAGE=1
  fi
  if [ -z "${CHART+set}" ] && [ -z "${NSM_PURGE+set}" ]; then
    echo "One of --chart or --all is required."
    SHOW_USAGE=1
  fi
  if [ -n "${CHART+set}" ] && [ -n "${NSM_PURGE}" ]; then
    echo "--chart and --all cannot be used together."
    SHOW_USAGE=1
  fi
  if [ $SHOW_USAGE -ne 0 ]; then
    usage
    exit 1
  fi
}


with_helm2() {
  if [ -z ${NSM_PURGE+x} ]; then
    $HELM delete --purge "${CHART}"
  else
    $HELM list --namespace="${NSM_NAMESPACE}" --short | xargs --no-run-if-empty -L1 helm delete --purge
  fi
}

with_helm3() {
  if [ -z ${NSM_PURGE+x} ]; then
    $HELM uninstall -n "$NSM_NAMESPACE" "${CHART}"
  else
    $HELM list -n "${NSM_NAMESPACE}" --short | xargs --no-run-if-empty -L1 helm uninstall -n "${NSM_NAMESPACE}"
  fi
}

while [[ $# -gt 0 ]]
do
key="$1"
case $key in
    --chart)
    CHART="$2"
    shift
    shift
    ;;
    --nsm_namespace)
    NSM_NAMESPACE="$2"
    shift
    shift
    ;;
    --all)
    NSM_PURGE=true
    shift
    ;;
    -h|--help)
    usage
    exit
    ;;
    *)
    shift
    ;;
esac
done

[ -n "$HELM" ] || HELM=helm

if ! command -v $HELM > /dev/null; then
  echo "Unable to locate Helm client '$HELM'"
  exit 1
fi

echo
if [ -z "$HELM_VERSION" ]; then
  HELM_VERSION=$($HELM version 2> /dev/null | head -1 | awk -v FS="(Ver\"|\")" '{print$ 2}')
  echo "Helm version detected: $HELM_VERSION"
else
  echo "Using Helm '$HELM_VERSION'"
fi

check_flags

echo "Cleaning up NSM"

if [[ $HELM_VERSION = v2* ]]
then
  with_helm2
elif [[ $HELM_VERSION = v3* ]]
then
  with_helm3
else
  echo "Unsupported helm version: $HELM_VERSION"
fi
