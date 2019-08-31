#!/usr/bin/env bash

echo "Running delete-nsm-namespaces script"
echo "NSM_NAMESPACE=${NSM_NAMESPACE}"
for ns in $(kubectl get ns -o custom-columns=NAME:.metadata.name); do
  if [[ ${ns} == NAME ]]; then
    continue
  fi
  echo "Checking ns: ${ns}"
  if [[ ${ns} == *${NSM_NAMESPACE}* ]]; then
    echo "Deleting ns: ${ns}"
    kubectl delete ns "${ns}"
    continue
  fi
done