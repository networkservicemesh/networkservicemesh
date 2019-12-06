# Copyright 2019 VMware, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at:
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

CHARTS=$(shell ls deployments/helm)
INSTALL_CHARTS=$(addprefix helm-install-,$(CHARTS))
DELETE_CHARTS=$(addprefix helm-delete-,$(CHARTS))
HELM_TIMEOUT?=300

# .PHONY: helm-init
# helm-init:
# 	helm init --wait && ./scripts/helm-patch-tiller.sh

# workaround for https://github.com/helm/helm/issues/6361:
#     helm with versions newer 2.15.x and older 3.x fails in case of one of apiservices is not available
#     that's why we downgrade it to 2.14 until PR for updating to 3.x will be ready
# workaround for https://github.com/helm/helm/issues/6374:
#     helm with versions older than 2.15.x is not working on Kubernetes 1.16
#     that's why we hack helm-init target
.PHONY: helm-init
helm-init:
	helm init --service-account tiller --override \
	spec.selector.matchLabels.'name'='tiller',spec.selector.matchLabels.'app'='helm' --output yaml \
	| sed 's@apiVersion: extensions/v1beta1@apiVersion: apps/v1@' \
	| kubectl apply -f - && kubectl wait -n kube-system --timeout=150s --for condition=Ready pod -l app=helm -l name=tiller

.PHONY: $(INSTALL_CHARTS)
$(INSTALL_CHARTS): export CHART=$(subst helm-install-,,$@)
$(INSTALL_CHARTS):
	# We specifically set admission-webhook variables here as it is a subchart
	# there might be a way to set these as global and refer to them with .Values.global.org
	# but that seems more intrusive than this hack. Consider changing to global if the charts
	# get even more complicated
	helm install --name=${CHART} \
	--atomic --timeout ${HELM_TIMEOUT} \
	--set org="${CONTAINER_REPO}",tag="${CONTAINER_TAG}" \
	--set forwardingPlane="${FORWARDING_PLANE}" \
	--set insecure="${INSECURE}" \
	--set global.JaegerTracing="true" \
	--set spire.enabled="${SPIRE_ENABLED}",spire.org="${CONTAINER_REPO}",spire.tag="${CONTAINER_TAG}" \
	--set admission-webhook.org="${CONTAINER_REPO}",admission-webhook.tag="${CONTAINER_TAG}" \
	--set prefix-service.org="${CONTAINER_REPO}",prefix-service.tag="${CONTAINER_TAG}" \
	--namespace="${NSM_NAMESPACE}" \
	deployments/helm/${CHART}

.PHONY: $(DELETE_CHARTS)
$(DELETE_CHARTS): export CHART=$(subst helm-delete-,,$@)
$(DELETE_CHARTS):
	helm delete --purge ${CHART} || true
