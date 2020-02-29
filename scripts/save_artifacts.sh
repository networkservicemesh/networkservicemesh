#!/bin/bash

if [[ "$1" == "only-master" ]] && [[ ${ARTIFACTS_SAVE_ALWAYS} != true ]] ; then
  echo "Logs not saved: env(${ARTIFACTS_SAVE_ALWAYS}) is not true"
  exit 0
fi

path=$(realpath "$0")
dir=$(dirname "$path")
go run "${dir}/../test/tools/k8s-artifacts-exporter/main.go"