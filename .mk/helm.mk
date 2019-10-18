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

.PHONY: helm-init
helm-init:
	helm init --wait && ./scripts/helm-patch-tiller.sh

.PHONY: $(INSTALL_CHARTS)
$(INSTALL_CHARTS): export CHART=$(subst helm-install-,,$@)
$(INSTALL_CHARTS):
	# We specifically set admission-webhook variables here as it is a subchart
	# there might be a way to set these as global and refer to them with .Values.global.org
	# but that seems more intrusive than this hack. Consider changing to global if the charts
	# get even more complicated
	helm install --name=${CHART} \
	--wait --timeout 300 \
	--set org="${CONTAINER_REPO}",tag="${CONTAINER_TAG}" \
	--set forwardingPlane="${FORWARDING_PLANE}" \
	--set insecure="false" \
	--set global.JaegerTracing="true" \
	--set spire.enabled="${SPIRE_ENABLED}",spire.org="${CONTAINER_REPO}",spire.tag="${CONTAINER_TAG}" \
	--set admission-webhook.org="${CONTAINER_REPO}",admission-webhook.tag="${CONTAINER_TAG}" \
	--namespace="${NSM_NAMESPACE}" \
	deployments/helm/${CHART}

.PHONY: $(DELETE_CHARTS)
$(DELETE_CHARTS): export CHART=$(subst helm-delete-,,$@)
$(DELETE_CHARTS):
	helm delete --purge ${CHART} || true
