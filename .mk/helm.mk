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
INSECURE?=false
PROMETHEUS?=true
METRICS_COLLECTOR_ENABLED?=true

.PHONY: helm-init
helm-init:
	./scripts/helm-init-wrapper.sh

.PHONY: $(INSTALL_CHARTS)
$(INSTALL_CHARTS): export CHART=$(subst helm-install-,,$@)
$(INSTALL_CHARTS):
	./scripts/helm-nsm-install.sh --chart ${CHART} \
	--container_repo ${CONTAINER_REPO} \
	--container_tag ${CONTAINER_TAG} \
	--forwarding_plane ${FORWARDING_PLANE} \
	--insecure ${INSECURE} \
	--networkservice "${NETWORK_SERVICE}" \
	--enable_prometheus ${PROMETHEUS} \
	--enable_metric_collection ${METRICS_COLLECTOR_ENABLED} \
	--nsm_namespace ${NSM_NAMESPACE}

.PHONY: $(DELETE_CHARTS)
$(DELETE_CHARTS): export CHART=$(subst helm-delete-,,$@)
$(DELETE_CHARTS):
	./scripts/helm-nsm-uninstall.sh --nsm_namespace ${NSM_NAMESPACE} --chart ${CHART} || true

.PHONY: helm-delete
helm-delete:
	./scripts/helm-nsm-uninstall.sh --nsm_namespace ${NSM_NAMESPACE} --all
