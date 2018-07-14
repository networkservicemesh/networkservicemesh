#!/bin/bash

# Copyright (c) 2018 Cisco and/or its affiliates.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

## Sample test scripts for adding sidecar components in a Kubernetes cluster
SIDECAR_CONFIG=conf/sidecar-injector

## Create SSL certificates
$SIDECAR_CONFIG/webhook-create-signed-cert.sh --service sidecar-injector-webhook-svc --secret sidecar-injector-webhook-certs --namespace default

## Copy the cert to the webhook configuration YAML file
< $SIDECAR_CONFIG/mutatingWebhookConfiguration.yaml $SIDECAR_CONFIG/webhook-patch-ca-bundle.sh >  $SIDECAR_CONFIG/mutatingwebhook-ca-bundle.yaml

## Create all the required components
kubectl create -f $SIDECAR_CONFIG/configMap.yaml -f $SIDECAR_CONFIG/ServiceAccount.yaml -f $SIDECAR_CONFIG/server-deployment.yaml -f $SIDECAR_CONFIG/mutatingwebhook-ca-bundle.yaml -f $SIDECAR_CONFIG/sidecarInjectorService.yaml

