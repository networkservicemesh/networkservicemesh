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

SPIRE_DIR = k8s/conf/spire
SPIRE_NAMESPACE_CONF = spire-namespace
SPIRE_SERVER = server-account server-configmap server-secrets server-service spire-roles server-statefulset
SPIRE_AGENT = agent-account agent-configmap agent-daemonset
SPIRE_AGENT_AZURE = agent-account agent-configmap-azure agent-daemonset

.PHONY: spire-start
spire-start: spire-server-start spire-agent-start

.PHONY: spire-delete
spire-delete: spire-agent-delete spire-server-delete

.PHONY: spire-namespace-create
spire-namespace-create: $(addsuffix -apply,$(addprefix spire-,$(SPIRE_NAMESPACE_CONF)))

.PHONY: spire-namespace-delete
spire-namespace-delete: $(addsuffix -delete,$(addprefix spire-,$(SPIRE_NAMESPACE_CONF)))

.PHONY: spire-server-start
spire-server-start: spire-namespace-create $(addsuffix -apply,$(addprefix spire-,$(SPIRE_SERVER)))

.PHONY: spire-agent-start
spire-agent-start: spire-namespace-create $(addsuffix -apply,$(addprefix spire-,$(SPIRE_AGENT)))

.PHONY: spire-server-delete
spire-server-delete: $(addsuffix -delete,$(addprefix spire-,$(SPIRE_SERVER))) spire-namespace-delete

.PHONY: spire-agent-delete
spire-agent-delete: $(addsuffix -delete,$(addprefix spire-,$(SPIRE_AGENT))) spire-namespace-delete

.PHONY: spire-%-apply
spire-%-apply:
	@sed "s;\(image:[ \t]*\)\(networkservicemesh\)\(/[^:]*\).*;\1${CONTAINER_REPO}\3$${COMMIT/$${COMMIT}/:$${COMMIT}};" ${SPIRE_DIR}/$*.yaml | $(kubectl) apply -n spire -f -

.PHONY: spire-%-delete
spire-%-delete:
	@echo "Deleting ${SPIRE_DIR}/$*.yaml"
	@$(kubectl) delete -n spire -f ${SPIRE_DIR}/$*.yaml > /dev/null 2>&1 || echo "$* does not exist and thus cannot be deleted"

.PHONY: spire-agent-start-azure
spire-agent-start-azure: spire-namespace-create $(addsuffix -apply,$(addprefix spire-,$(SPIRE_AGENT_AZURE)))
