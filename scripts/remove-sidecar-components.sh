#!/bin/bash


## Create all the required components
kubectl delete -f conf/sample/sidecar-injector/configMap.yaml -f conf/sample/sidecar-injector/ServiceAccount.yaml -f conf/sample/sidecar-injector/server-deployment.yaml -f conf/sample/sidecar-injector/mutatingwebhook-ca-bundle.yaml -f conf/sample/sidecar-injector/sidecarInjectorService.yaml

