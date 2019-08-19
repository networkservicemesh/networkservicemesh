#!/bin/bash

set -x

kubectl create serviceaccount --namespace kube-system tiller
kubectl create clusterrolebinding tiller-cluster-rule --clusterrole=cluster-admin --serviceaccount=kube-system:tiller
kubectl patch deploy --namespace kube-system tiller-deploy -p "{\"spec\":{\"template\":{\"spec\":{\"serviceAccount\":\"tiller\"}}}}"
tillerPod=$(kubectl get pod -l app=helm -n kube-system | grep -v Terminating | awk 'NR>1 {print $1}')
kubectl wait -n kube-system --timeout=150s --for condition=Ready pods/"${tillerPod}"