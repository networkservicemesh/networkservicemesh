# Copyright (c) 2018 Cisco and/or its affiliates.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at:
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

K8S_CONF_DIR = k8s/conf/

# Need nsmdp and icmp-responder-nse here as well, but missing yaml files
DEPLOYS = nsmd icmp-responder-nse vppagent-dataplane nsc

CLUSTER_CONFIGS = cluster-role-admin cluster-role-binding cluster-role-view

# All of the rules that use vagrant are intentionally written in such a way
# That you could set the CLUSTER_RULES_PREFIX different and introduce
# a new platform to run on with k8s by adding a new include ${method}.mk
# and setting CLUSTER_RULES_PREFIX to a different value
ifeq ($(CLUSTER_RULES_PREFIX),)
CLUSTER_RULES_PREFIX := vagrant
endif
include .vagrant.mk

include .kubeadm.mk

# .null.mk allows you to skip the vagrant machinery with:
# export CLUSTER_RULES_PREFIX=null
# before running make
include .null.mk

# Pull in docker targets
CONTAINER_BUILD_PREFIX = docker
include .docker.mk

.PHONY: k8s-deploy
k8s-deploy: k8s-delete $(addsuffix -deploy,$(addprefix k8s-,$(DEPLOYS)))

.PHONY: k8s-%-deploy
k8s-%-deploy:  k8s-start k8s-config k8s-%-delete k8s-%-load-images
	@kubectl apply -f ${K8S_CONF_DIR}/$*.yaml

.PHONY: k8s-delete
k8s-delete: $(addsuffix -delete,$(addprefix k8s-,$(DEPLOYS)))

.PHONY: k8s-%-delete
k8s-%-delete:
	@echo "Deleting ${K8S_CONF_DIR}/$*.yaml"
	@kubectl delete -f ${K8S_CONF_DIR}/$*.yaml > /dev/null 2>&1 || echo "$* does not exist and thus cannot be deleted"

.PHONY: k8s-load-images
k8s-load-images: $(addsuffix -load-images,$(addprefix k8s-,$(DEPLOYS)))

.PHONY: k8s-%-load-images
k8s-%-load-images:  k8s-start $(CLUSTER_RULES_PREFIX)-%-load-images
	@echo "Delegated to $(CLUSTER_RULES_PREFIX)-$*-load-images"

.PHONY: k8s-%-config
k8s-%-config:  k8s-start
	@kubectl apply -f ${K8S_CONF_DIR}/$*.yaml

.PHONY: k8s-config
k8s-config: $(addsuffix -config,$(addprefix k8s-,$(CLUSTER_CONFIGS)))

.PHONY: k8s-start
k8s-start: $(CLUSTER_RULES_PREFIX)-start

.PHONY: k8s-start
k8s-restart: $(CLUSTER_RULES_PREFIX)-restart

.PHONY: k8s-build
k8s-build: $(addsuffix -build,$(addprefix k8s-,$(DEPLOYS)))

.PHONY: k8s-save-deploy
k8s-save-deploy: k8s-delete $(addsuffix -save-deploy,$(addprefix k8s-,$(DEPLOYS)))

.PHONY: k8s-%-save-deploy
k8s-%-save-deploy:  k8s-start k8s-config k8s-%-save  k8s-%-load-images
	@kubectl apply -f ${K8S_CONF_DIR}/$*.yaml

NSMD_CONTAINERS = nsmd nsmdp nsmd-k8s
.PHONY: k8s-nsmd-build
k8s-nsmd-build:  $(addsuffix -build,$(addprefix ${CONTAINER_BUILD_PREFIX}-,$(NSMD_CONTAINERS)))

.PHONY: k8s-nsmd-save
k8s-nsmd-save:  $(addsuffix -save,$(addprefix ${CONTAINER_BUILD_PREFIX}-,$(NSMD_CONTAINERS)))

.PHONY: k8s-nsmd-load-images
k8s-nsmd-load-images:  k8s-start $(addsuffix -load-images,$(addprefix ${CLUSTER_RULES_PREFIX}-,$(NSMD_CONTAINERS)))

VPPAGENT_DATAPLANE_CONTAINERS = vppagent vppagent-dataplane
.PHONY: k8s-vppagent-dataplane-build
k8s-vppagent-dataplane-build:  $(addsuffix -build,$(addprefix ${CONTAINER_BUILD_PREFIX}-,$(VPPAGENT_DATAPLANE_CONTAINERS)))

.PHONY: k8s-vppagent-dataplane-save
k8s-vppagent-dataplane-save:  $(addsuffix -save,$(addprefix ${CONTAINER_BUILD_PREFIX}-,$(VPPAGENT_DATAPLANE_CONTAINERS)))

.PHONY: k8s-vppagent-dataplane-load-images
k8s-vppagent-dataplane-load-images:  k8s-start $(addsuffix -load-images,$(addprefix ${CLUSTER_RULES_PREFIX}-,$(VPPAGENT_DATAPLANE_CONTAINERS)))

.PHONY: k8s-nsc-build
k8s-nsc-build:  ${CONTAINER_BUILD_PREFIX}-nsc-build

.PHONY: k8s-nsc-save
k8s-nsc-save:  ${CONTAINER_BUILD_PREFIX}-nsc-save

.PHONY: k8s-icmp-responder-nse-build
k8s-icmp-responder-nse-build:  ${CONTAINER_BUILD_PREFIX}-icmp-responder-nse-build

.PHONY: k8s-icmp-responder-nse-save
k8s-icmp-responder-nse-save:  ${CONTAINER_BUILD_PREFIX}-icmp-responder-nse-save

# TODO add k8s-%-logs and k8s-logs to capture all the logs from k8s