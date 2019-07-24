# Copyright (c) 2018 Cisco and/or its affiliates.
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

NSM_NAMESPACE = `cat "${K8S_CONF_DIR}/${CLUSTER_CONFIG_NAMESPACE}.yaml" | awk '/name:/ {print $$2}'`
K8S_CONF_DIR = k8s/conf

.PHONY: spire-start
spire-start: spire-server-start spire-agent-start

.PHONY: spire-delete
spire-delete: spire-agent-delete spire-server-delete

.PHONY: spire-server-start
spire-server-start:
	@echo "Starting spire-server...";
	@kubectl apply -f ${K8S_CONF_DIR}/spire/spire-namespace.yaml;
	@kubectl apply -f ${K8S_CONF_DIR}/spire/server-account.yaml;
	@kubectl apply -f ${K8S_CONF_DIR}/spire/server-configmap.yaml;
	@kubectl apply -f ${K8S_CONF_DIR}/spire/server-secrets.yaml;
	@kubectl apply -f ${K8S_CONF_DIR}/spire/server-service.yaml;
	@kubectl apply -f ${K8S_CONF_DIR}/spire/spire-roles.yaml;
	@kubectl apply -f ${K8S_CONF_DIR}/spire/server-statefulset.yaml;

.PHONY: spire-agent-start
spire-agent-start:
	@echo "Starting spire-agent...";
	@kubectl apply -f ${K8S_CONF_DIR}/spire/agent-account.yaml;
	@kubectl apply -f ${K8S_CONF_DIR}/spire/agent-configmap.yaml;
	@kubectl apply -f ${K8S_CONF_DIR}/spire/agent-daemonset.yaml;


.PHONY: spire-agent-start-azure
spire-agent-start-azure:
	@echo "Starting spire-agent...";
	@kubectl apply -f ${K8S_CONF_DIR}/spire/agent-account.yaml;
	@kubectl apply -f ${K8S_CONF_DIR}/spire/agent-configmap-azure.yaml;
	@kubectl apply -f ${K8S_CONF_DIR}/spire/agent-daemonset.yaml;

.PHONY: spire-server-delete
spire-server-delete:
	@echo "Deleting spire-server...";
	@kubectl delete -f ${K8S_CONF_DIR}/spire/server-statefulset.yaml || \
	kubectl delete -f ${K8S_CONF_DIR}/spire/spire-roles.yaml || \
	kubectl delete -f ${K8S_CONF_DIR}/spire/server-account.yaml || \
	kubectl delete -f ${K8S_CONF_DIR}/spire/server-configmap.yaml || \
	kubectl delete -f ${K8S_CONF_DIR}/spire/server-secrets.yaml || \
	kubectl delete -f ${K8S_CONF_DIR}/spire/server-service.yaml || \
	kubectl delete -f ${K8S_CONF_DIR}/spire/spire-namespace.yaml;

.PHONY: spire-agent-delete
spire-agent-delete:
	@echo "Deleting spire-agent...";
	@kubectl delete -f ${K8S_CONF_DIR}/spire/agent-configmap.yaml;
	@kubectl delete -f ${K8S_CONF_DIR}/spire/agent-account.yaml;
	@kubectl delete -f ${K8S_CONF_DIR}/spire/agent-daemonset.yaml;