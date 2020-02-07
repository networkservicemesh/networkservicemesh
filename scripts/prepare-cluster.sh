#!/bin/bash

# workaround for https://github.com/helm/helm/issues/6361:
function delete_unavailable_apiservice() {
  for service in $(kubectl get apiservice | grep False | awk '{print $1}'); do
    echo "Deleting ${service}..."
    kubectl delete apiservice "${service}"
  done
}

function has_unavailable_apiservice() {
  kubectl get apiservice | grep False
}

if has_unavailable_apiservice; then
  echo "Unavailable Kubernetes services detected, trying to wait when there are none..."
  if timeout 180s bash -c "while kubectl get apiservice | grep -q False; do sleep 10; echo 'tick'; done"; then
    echo "Success!"
  else
    echo "There still might be some services left:"
    has_unavailable_apiservice
    echo "Last resort: removing unavailable services.."
    delete_unavailable_apiservice
  fi
fi

echo "Preparing cluster..."
make helm-init k8s-config spire-install
ERR=$?

echo
echo "Cluster info ====================================================="
kubectl get all -A -o wide

echo
if [ "$ERR" == 0 ]; then
  echo "Cluster prepare is OK"
else
  echo "Cluster prepare failed"
fi

exit $ERR
