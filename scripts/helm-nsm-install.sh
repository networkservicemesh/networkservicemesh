#!/bin/bash

# A script for wrapping NSM installation via Helm. Both Helm 2 and Helm 3 are
# supported.


function with_helm_2 () {
  # We specifically set admission-webhook variables here as it is a subchart
  # there might be a way to set these as global and refer to them with .Values.global.org
  # but that seems more intrusive than this hack. Consider changing to global if the charts
  # get even more complicated
  echo -n "Performing chart installation..."
  helm install --name="$CHART" \
  --atomic --timeout 300 \
  --set org="$CONTAINER_REPO",tag="$CONTAINER_TAG" \
  --set forwardingPlane="$FORWARDING_PLANE" \
  --set insecure="$INSECURE" \
  --set networkservice="${NETWORK_SERVICE}" \
  --set prometheus="${PROMETHEUS}" \
  --set metricsCollectorEnabled="${METRICS_COLLECTOR_ENABLED}" \
  --set global.JaegerTracing="true" \
  --set spire.enabled="$SPIRE_ENABLED",spire.org="$CONTAINER_REPO",spire.tag="$CONTAINER_TAG" \
  --set admission-webhook.org="$CONTAINER_REPO",admission-webhook.tag="$CONTAINER_TAG" \
  --set prefix-service.org="$CONTAINER_REPO",prefix-service.tag="$CONTAINER_TAG" \
  --namespace="$NSM_NAMESPACE" \
  deployments/helm/"$CHART" &
  PID=$!
  sleep 2
  while ps -p ${PID} > /dev/null;
  do
    printf "."
    sleep 2
  done
  echo "Done"
}

function with_helm_3 () {
  # Ensure we have the nsm namespace available
  echo -n "Ensure namespace $NSM_NAMESPACE exists..."
  kubectl apply -f k8s/conf/namespace-nsm.yaml &
  PID=$!
  sleep 2
  while ps -p ${PID} > /dev/null;
  do
    printf "."
    sleep 2
  done
  echo "Done"

  # We specifically set admission-webhook variables here as it is a subchart
  # there might be a way to set these as global and refer to them with .Values.global.org
  # but that seems more intrusive than this hack. Consider changing to global if the charts
  # get even more complicated
  echo -n "Performing chart installation..."
  helm install "$CHART" \
  --atomic --timeout 5m \
  --set org="$CONTAINER_REPO",tag="$CONTAINER_TAG" \
  --set forwardingPlane="$FORWARDING_PLANE" \
  --set insecure="$INSECURE" \
  --set networkservice="${NETWORK_SERVICE}" \
  --set prometheus="${PROMETHEUS}" \
  --set metricsCollectorEnabled="${METRICS_COLLECTOR_ENABLED}" \
  --set global.JaegerTracing="true" \
  --set spire.enabled="$SPIRE_ENABLED",spire.org="$CONTAINER_REPO",spire.tag="$CONTAINER_TAG" \
  --set admission-webhook.org="$CONTAINER_REPO",admission-webhook.tag="$CONTAINER_TAG" \
  --set prefix-service.org="$CONTAINER_REPO",prefix-service.tag="$CONTAINER_TAG" \
  --namespace "$NSM_NAMESPACE" \
  deployments/helm/"$CHART" &
  PID=$!
  sleep 2
  while ps -p ${PID} > /dev/null;
  do
    printf "."
    sleep 2
  done
  echo "Done"
}

function usage () {
  echo "Usage: $0 [flags]"
  echo "Available flags:"
  FLAGS="--chart [chart],Chart name. Defaults to value of CHART
--container_repo [repo URI],Container repository. Defaults to value of CONTAINER_REPO
--container_tag [tag],Container tag. Defaults to value of CONTAINER_TAG
--forwarding_plane [forwarding plane],NSM forwarding plane to use. Defaults to value of FORWARDING_PLANE
--insecure [true|false],Defaults to value of INSECURE
--networkservice [endpoint network service],Name of Network Service connect to.
--nsm_namespace [namespace],Name of the NSM namespace. Defaults to value of NSM_NAMESPACE
--spire_enabled [true|false],Defaults to value of SPIRE_ENABLED
-h | --help,Display usage"

echo -e "$FLAGS" | column -t --separator ","
}

function check_flags () {
  SHOW_USAGE=0
  if [ -z ${CHART+x} ]; then
    echo "CHART is not set, use --chart or set a value for it."
    SHOW_USAGE=1
  fi
  if [ -z ${CONTAINER_REPO+x} ]; then
    echo "CONTAINER_REPO is not set, use --container_repo or set a value for it."
    SHOW_USAGE=1
  fi
  if [ -z ${CONTAINER_TAG+x} ]; then
    echo "CONTAINER_TAG is not set, use --container_tag or set a value for it."
    SHOW_USAGE=1
  fi
  if [ -z ${INSECURE+x} ]; then
    echo "INSECURE is not set, use --insecure or set a value for it."
    SHOW_USAGE=1
  fi
  if [ -z ${NETWORK_SERVICE+x} ]; then
    echo "NETWORK_SERVICE is not set, use --networkservice or set a value for it."
    SHOW_USAGE=1
  fi
  if [ -z ${FORWARDING_PLANE+x} ]; then
    echo "FORWARDING_PLANE is not set, use --forwarding_plane or set a value for it."
    SHOW_USAGE=1
  fi
  if [ -z ${NSM_NAMESPACE+x} ]; then
    echo "NSM_NAMESPACE is not set, use --nsm_namespace or set a value for it."
    SHOW_USAGE=1
  fi
  if [ -z ${PROMETHEUS+x} ]; then
    echo "PROMETHEUS is not set, use --enable_prometheus or set a value for it."
    SHOW_USAGE=1
  fi
  if [ -z ${METRICS_COLLECTOR_ENABLED+x} ]; then
    echo "METRICS_COLLECTOR_ENABLED is not set, use --enable_metric_collection or set a value for it."
    SHOW_USAGE=1
  fi
  if [ $SHOW_USAGE -ne 0 ]; then
    usage
    exit 1
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
    --container_repo)
    CONTAINER_REPO="$2"
    shift
    shift
    ;;
    --container_tag)
    CONTAINER_TAG="$2"
    shift
    shift
    ;;
    --insecure)
    INSECURE="$2"
    shift
    shift
    ;;
    --networkservice)
    NETWORK_SERVICE="$2"
    shift
    shift
    ;;
    --forwarding_plane)
    FORWARDING_PLANE="$2"
    shift
    shift
    ;;
    --nsm_namespace)
    NSM_NAMESPACE="$2"
    shift
    shift
    ;;
    --enable_prometheus)
    PROMETHEUS="$2"
    shift
    shift
    ;;
    --enable_metric_collection)
    METRICS_COLLECTOR_ENABLED="$2"
    shift
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


if ! command -v helm > /dev/null; then
  echo "Unable to locate helm client"
  exit 1
fi

HELM_VERSION=$(helm version 2> /dev/null | awk -v FS="(Ver\"|\")" '{print$ 2}')
check_flags

if [[ $HELM_VERSION = v2* ]]
then
  with_helm_2
elif [[ $HELM_VERSION = v3* ]]
then
  with_helm_3
else
  echo "Unsupported helm version: $HELM_VERSION"
fi
