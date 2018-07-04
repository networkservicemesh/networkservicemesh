#!/bin/bash


## Create SSL certificates
conf/sample/sidecar-injector/webhook-create-signed-cert.sh --service sidecar-injector-webhook-svc --secret sidecar-injector-webhook-certs --namespace default

## Copy the cert to the webhook configuration YAML file
cat conf/sample/sidecar-injector/mutatingWebhookConfiguration.yaml | conf/sample/sidecar-injector/webhook-patch-ca-bundle.sh >  conf/sample/sidecar-injector/mutatingwebhook-ca-bundle.yaml

## Create all the required components
kubectl create -f conf/sample/sidecar-injector/configMap.yaml -f conf/sample/sidecar-injector/ServiceAccount.yaml -f conf/sample/sidecar-injector/server-deployment.yaml -f conf/sample/sidecar-injector/mutatingwebhook-ca-bundle.yaml -f conf/sample/sidecar-injector/sidecarInjectorService.yaml

