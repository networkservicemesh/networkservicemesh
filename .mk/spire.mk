# Copyright (c) 2019 Cisco and/or its affiliates.
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

SPIRE_NAMESPACE=spire

.PHONY: spire-install
spire-install:
	helm install --name=spire \
	--wait --timeout 300 \
	--set org="${CONTAINER_REPO}",tag="${CONTAINER_TAG}" \
	--namespace="${SPIRE_NAMESPACE}" \
	deployments/helm/nsm/charts/spire

# temporary workaround for azure
.PHONY: spire-install-azure
spire-install-azure:
	helm install --name=spire \
	--wait --timeout 300 \
	--set org="${CONTAINER_REPO}",tag="${CONTAINER_TAG}" \
	--set azure.enabled=true \
	--namespace="${SPIRE_NAMESPACE}" \
	deployments/helm/nsm/charts/spire

.PHONY: spire-delete
spire-delete:
	helm delete --purge spire
