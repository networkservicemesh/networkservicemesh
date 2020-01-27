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

NSM_NAMESPACE?=nsm-system

CONTAINER_REPO?=networkservicemesh
CONTAINER_TAG?=master

NSMGR_CONTAINERS := nsmd nsmd-k8s nsmdp

# All of the rules that use vagrant are intentionally written in such a way
# That you could set the CLUSTER_RULES_PREFIX different and introduce
# a new platform to run on with k8s by adding a new include ${method}.mk
# and setting CLUSTER_RULES_PREFIX to a different value
CLUSTER_RULES_PREFIX ?= kind
include .mk/kind.mk
include .mk/vagrant.mk
include .mk/packet.mk
include .mk/aws.mk
include .mk/azure.mk

# .null.mk allows you to skip the vagrant machinery with:
# export CLUSTER_RULES_PREFIX=null
# before running make
include .mk/null.mk

include .mk/gke.mk

SPIRE_ENABLED?=true
IMAGE_DIR=build/images/

kubectl = kubectl -n ${NSM_NAMESPACE}
images_tar = $(subst .tar,,$(filter %.tar, $(shell mkdir -p ./build/images;ls ./build/images)))

export ORG=$(CONTAINER_REPO)

.PHONY: k8s-load-images
k8s-load-images: $(addsuffix -load-images,$(addprefix k8s-,$(images_tar)))

.PHONY: k8s-%-load-images
k8s-%-load-images:  k8s-start $(CLUSTER_RULES_PREFIX)-%-load-images
	@echo "Delegated to $(CLUSTER_RULES_PREFIX)-$*-load-images"

.PHONY: k8s-config
k8s-config: helm-install-config

.PHONY: k8s-deconfig
k8s-deconfig: helm-delete-config

.PHONY: k8s-start
k8s-start: $(CLUSTER_RULES_PREFIX)-start

.PHONY: k8s-restart
k8s-restart: $(CLUSTER_RULES_PREFIX)-restart

.PHONY: k8s-build
k8s-build: docker-build

.PHONY: k8s-save
k8s-save: docker-save

.PHONY: k8s-delete-nsm-namespaces
k8s-delete-nsm-namespaces:
	@NSM_NAMESPACE=${NSM_NAMESPACE} ./scripts/delete-nsm-namespaces.sh

.PHONY: k8s-%logs
k8s-%-logs:
	@echo "K8s logs for $*"
	@for pod in $$($(kubectl) get pods --all-namespaces | grep $* | awk '{print $$2}');do \
		echo '******************************************'; \
		echo "Logs: $${pod}:"; \
		$(kubectl) logs $${pod} || true; \
		$(kubectl) logs -p $${pod} || true; \
		echo '>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>'; \
		echo "Network information for $${pod}"; \
		$(kubectl) exec -ti $${pod} ip addr; \
		$(kubectl) exec -ti $${pod} ip neigh; \
		if [[ "$${pod}" == *"vppagent"* ]]; then \
			echo "vpp information for $${pod}"; \
			$(kubectl) exec -it $${pod} vppctl show int; \
			$(kubectl) exec -it $${pod} vppctl show int addr; \
			$(kubectl) exec -it $${pod} vppctl show vxlan tunnel; \
			$(kubectl) exec -it $${pod} vppctl show memif; \
		fi; \
	done

.PHONY: k8s-nsmgr-logs
k8s-nsmgr-logs:
	@echo "K8s logs for nsmds"
	@echo '******************************************'
	@for pod in $$($(kubectl) get pods --all-namespaces | grep nsmgr | awk '{print $$2}'); do \
		for container in nsmd nsmdp nsmd-k8s; do \
			echo '------------------------------------------'; \
			echo "K8s logs for $${pod} container $${container}"; \
			$(kubectl) logs $${pod} --container $${container} || true; \
			$(kubectl) logs -p $${pod} --container $${container} || true ;\
			echo '>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>'; \
			echo 'NSMD Network information'; \
			$(kubectl) exec -ti $${pod} --container $${container} ip addr; \
		done \
	done

.PHONY: k8s-%-debug
k8s-%-debug:
	@echo "Debugging $*"
	@$(kubectl) exec -ti $$($(kubectl) get pods | grep $*- | cut -d \  -f1) /go/src/github.com/networkservicemesh/networkservicemesh/scripts/debug.sh $*

.PHONY: k8s-nsmgr-debug
k8s-nsmgr-debug:
	@$(kubectl) exec -ti $(pod) -c nsmd /go/src/github.com/networkservicemesh/networkservicemesh/scripts/debug.sh nsmd

.PHONY: k8s-forward
k8s-forward:
	@echo "Forwarding local $(port1) to $(port2) for $(pod)"
	@$(kubectl) port-forward $$($(kubectl) get pods | grep $(pod) | cut -d \  -f1) $(port1):$(port2)

.PHONY: k8s-check
k8s-check: k8s-icmp-check k8s-vpn-check

.PHONY: k8s-icmp-check
k8s-icmp-check:
	$(info Checking icmp...)
	@NSM_NAMESPACE=${NSM_NAMESPACE} ./scripts/nsc_ping_all.sh

.PHONY: k8s-vpn-check
k8s-vpn-check:
	$(info Checking vpn...)
	@NSM_NAMESPACE=${NSM_NAMESPACE} ./scripts/verify_vpn_gateway.sh

.PHONY: k8s-crossconnect-monitor-check
k8s-crossconnect-monitor-check:
	$(info Checking crossconnect-monitor...)
	@NSM_NAMESPACE=${NSM_NAMESPACE} ./scripts/verify_crossconnect_monitor.sh

.PHONY: k8s-logs-snapshot
k8s-logs-snapshot:
	@NSM_NAMESPACE=${NSM_NAMESPACE} ./scripts/logs_snapshot.sh

k8s-logs-snapshot-only-master:
	@NSM_NAMESPACE=${NSM_NAMESPACE} ./scripts/logs_snapshot.sh only-master

.PHONY: k8s-terminating-cleanup
k8s-terminating-cleanup:
	@$(kubectl) get pods -o wide |grep Terminating | cut -d \  -f 1 | xargs $(kubectl) delete pods --force --grace-period 0 {}

.PHONE: k8s-pods
k8s-pods:
	@$(kubectl) get pods -o wide

.PHONY: k8s-nsmgr-master-tlogs
k8s-nsmgr-master-tlogs:
	@$(kubectl) logs -f $$($(kubectl) get pods -o wide | grep kube-master | grep nsmgr | cut -d\  -f1) -c nsmd

.PHONY: k8s-nsmgr-worker-tlogs
k8s-nsmgr-worker-tlogs:
	@$(kubectl) logs -f $$($(kubectl) get pods -o wide | grep kube-worker | grep nsmgr | cut -d\  -f1) -c nsmd

.PHONY: k8s-nsmgr-build
k8s-nsmgr-build: $(addsuffix -build, $(addprefix docker-, nsmd nsmd-k8s nsmdp))

.PHONY: k8s-nsmgr-save
k8s-nsmgr-save: $(addsuffix -save, $(addprefix docker-, $(NSMGR_CONTAINERS)))

.PHONY: k8s-nsmgr-load-images
k8s-nsmgr-load-images: $(addsuffix -load-images, $(addprefix $(CLUSTER_RULES_PREFIX)-, $(NSMGR_CONTAINERS)))

.PHONY: k8s-nsmgr-update
k8s-nsmgr-update: k8s-nsmgr-save k8s-nsmgr-load-images

.PHONY: k8s-%-update
k8s-%-update: docker-%-save k8s-%-load-images
	$(info Image updated in cluster: $*)
