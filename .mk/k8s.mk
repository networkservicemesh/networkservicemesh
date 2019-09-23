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

K8S_CONF_DIR = k8s/conf

# Deployments - common
DEPLOY_TRACING = jaeger
DEPLOY_SIDECARS = nsm-init nsm-monitor nsm-coredns
DEPLOY_WEBHOOK = admission-webhook
DEPLOY_MONITOR = crossconnect-monitor skydive
DEPLOY_ICMP_KERNEL = icmp-responder-nse nsc
DEPLOY_ICMP = $(DEPLOY_ICMP_KERNEL)
# Set the configured forwarding plane
ifeq (${FORWARDING_PLANE}, vpp)
  # Deployments - VPP plane
  DEPLOY_FORWARDING_PLANE = vppagent-dataplane
  DEPLOY_ICMP_VPP = vppagent-icmp-responder-nse vppagent-nsc
  DEPLOY_VPN = secure-intranet-connectivity vppagent-firewall-nse vppagent-passthrough-nse vpn-gateway-nse vpn-gateway-nsc
  DEPLOY_ICMP += $(DEPLOY_ICMP_VPP)
else ifeq (${FORWARDING_PLANE}, kernel)
  # Deployments - Kernel plane
  DEPLOY_FORWARDING_PLANE = kernel-forwarder
endif
# Deployments - grouped
# Need nsmdp and icmp-responder-nse here as well, but missing yaml files
DEPLOY_NSM = nsmgr $(DEPLOY_FORWARDING_PLANE)
DEPLOY_PROXY_NSM = proxy-nsmgr
DEPLOY_INFRA = $(DEPLOY_TRACING) $(DEPLOY_WEBHOOK) $(DEPLOY_NSM) $(DEPLOY_PROXY_NSM) $(DEPLOY_MONITOR) $(DEPLOY_SIDECARS)
DEPLOYS = $(DEPLOY_INFRA) $(DEPLOY_ICMP) $(DEPLOY_VPN)

CLUSTER_CONFIG_ROLE = cluster-role-admin cluster-role-binding cluster-role-view
CLUSTER_CONFIG_CRD = crd-networkservices crd-networkserviceendpoints crd-networkservicemanagers
CLUSTER_CONFIG_NAMESPACE = namespace-nsm
CLUSTER_CONFIGS = $(CLUSTER_CONFIG_ROLE) $(CLUSTER_CONFIG_CRD) $(CLUSTER_CONFIG_NAMESPACE)

NSM_NAMESPACE = `cat "${K8S_CONF_DIR}/${CLUSTER_CONFIG_NAMESPACE}.yaml" | awk '/name:/ {print $$2}'`

# All of the rules that use vagrant are intentionally written in such a way
# That you could set the CLUSTER_RULES_PREFIX different and introduce
# a new platform to run on with k8s by adding a new include ${method}.mk
# and setting CLUSTER_RULES_PREFIX to a different value
ifeq ($(CLUSTER_RULES_PREFIX),)
CLUSTER_RULES_PREFIX := vagrant
endif
include .mk/vagrant.mk
include .mk/packet.mk
include .mk/aws.mk
include .mk/azure.mk

# .kind.mk enables the kind.sigs.k8s.io docker based K8s install:
# export CLUSTER_RULES_PREFIX=kind
# before running make
include .mk/kind.mk

# .null.mk allows you to skip the vagrant machinery with:
# export CLUSTER_RULES_PREFIX=null
# before running make
include .mk/null.mk

include .mk/docker.mk

include .mk/gke.mk

# Pull in docker targets
ifeq ($(CONTAINER_BUILD_PREFIX),)
CONTAINER_BUILD_PREFIX = docker
endif

ifeq ($(CONTAINER_BUILD_PREFIX),gcb)
include .mk/gcb.mk
CONTAINER_REPO=gcr.io/$(shell gcloud config get-value project)
CLUSTER_CONFIGS+=cpu-defaults
endif
ifeq ($(CONTAINER_REPO),)
CONTAINER_REPO=networkservicemesh
endif
ifeq ($(CONTAINER_TAG),)
CONTAINER_TAG=latest
endif

kubectl = kubectl -n ${NSM_NAMESPACE}

export ORG=$(CONTAINER_REPO)

.PHONY: k8s-load-images
k8s-load-images: $(addsuffix -load-images,$(addprefix k8s-,$(DEPLOYS)))

.PHONY: k8s-%-load-images
k8s-%-load-images:  k8s-start $(CLUSTER_RULES_PREFIX)-%-load-images
	@echo "Delegated to $(CLUSTER_RULES_PREFIX)-$*-load-images"

.PHONY: k8s-%-config
k8s-%-config:  k8s-start
	@$(kubectl) apply -f ${K8S_CONF_DIR}/$*.yaml

.PHONY: k8s-%-deconfig
k8s-%-deconfig:
	@$(kubectl) delete -f ${K8S_CONF_DIR}/$*.yaml || true

.PHONY: k8s-config
k8s-config: $(addsuffix -config,$(addprefix k8s-,$(CLUSTER_CONFIGS)))

.PHONY: k8s-deconfig
k8s-deconfig: $(addsuffix -deconfig,$(addprefix k8s-,$(CLUSTER_CONFIGS)))

.PHONY: k8s-start
k8s-start: $(CLUSTER_RULES_PREFIX)-start

.PHONY: k8s-restart
k8s-restart: $(CLUSTER_RULES_PREFIX)-restart

.PHONY: k8s-build
k8s-build: $(addsuffix -build,$(addprefix k8s-,$(DEPLOYS)))

.PHONY: k8s-nsm-coredns-save
k8s-nsm-coredns-save:  $(addsuffix -save,$(addprefix ${CONTAINER_BUILD_PREFIX}-,nsm-coredns))

.PHONY: k8s-nsm-coredns-build
k8s-nsm-coredns-build:  $(addsuffix -build,$(addprefix ${CONTAINER_BUILD_PREFIX}-,nsm-coredns))

.PHONY: k8s-jaeger-build
k8s-jaeger-build:

.PHONY: k8s-jaeger-save
k8s-jaeger-save:

.PHONY: k8s-jaeger-load-images
k8s-jaeger-load-images:

.PHONY: k8s-save
k8s-save: $(addsuffix -save,$(addprefix k8s-,$(DEPLOYS)))

NSMGR_CONTAINERS = nsmd nsmdp nsmd-k8s
.PHONY: k8s-nsmgr-build
k8s-nsmgr-build:  $(addsuffix -build,$(addprefix ${CONTAINER_BUILD_PREFIX}-,$(NSMGR_CONTAINERS)))

.PHONY: k8s-nsmgr-save
k8s-nsmgr-save:  $(addsuffix -save,$(addprefix ${CONTAINER_BUILD_PREFIX}-,$(NSMGR_CONTAINERS)))

.PHONY: k8s-nsmgr-load-images
k8s-nsmgr-load-images:  k8s-start $(addsuffix -load-images,$(addprefix ${CLUSTER_RULES_PREFIX}-,$(NSMGR_CONTAINERS)))

PROXY_NSMGR_CONTAINERS = proxy-nsmd proxy-nsmd-k8s
.PHONY: k8s-proxy-nsmgr-build
k8s-proxy-nsmgr-build:  $(addsuffix -build,$(addprefix ${CONTAINER_BUILD_PREFIX}-,$(PROXY_NSMGR_CONTAINERS)))

.PHONY: k8s-proxy-nsmgr-save
k8s-proxy-nsmgr-save:  $(addsuffix -save,$(addprefix ${CONTAINER_BUILD_PREFIX}-,$(PROXY_NSMGR_CONTAINERS)))

.PHONY: k8s-proxy-nsmgr-load-images
k8s-proxy-nsmgr-load-images:  $(addsuffix -load-images,$(addprefix ${CLUSTER_RULES_PREFIX}-,$(PROXY_NSMGR_CONTAINERS)))

VPPAGENT_DATAPLANE_CONTAINERS = vppagent-dataplane
.PHONY: k8s-vppagent-dataplane-build
k8s-vppagent-dataplane-build:  $(addsuffix -build,$(addprefix ${CONTAINER_BUILD_PREFIX}-,$(VPPAGENT_DATAPLANE_CONTAINERS)))
 .PHONY: k8s-vppagent-dataplane-save
k8s-vppagent-dataplane-save:  $(addsuffix -save,$(addprefix ${CONTAINER_BUILD_PREFIX}-,$(VPPAGENT_DATAPLANE_CONTAINERS)))
 .PHONY: k8s-vppagent-dataplane-load-images
k8s-vppagent-dataplane-load-images:  k8s-start $(addsuffix -load-images,$(addprefix ${CLUSTER_RULES_PREFIX}-,$(VPPAGENT_DATAPLANE_CONTAINERS)))

KERNEL_FORWARDER_CONTAINERS = kernel-forwarder
.PHONY: k8s-kernel-forwarder-build
k8s-kernel-forwarder-build:  $(addsuffix -build,$(addprefix ${CONTAINER_BUILD_PREFIX}-,$(KERNEL_FORWARDER_CONTAINERS)))
 .PHONY: k8s-kernel-forwarder-save
k8s-kernel-forwarder-save:  $(addsuffix -save,$(addprefix ${CONTAINER_BUILD_PREFIX}-,$(KERNEL_FORWARDER_CONTAINERS)))
 .PHONY: k8s-kernel-forwarder-load-images
k8s-kernel-forwarder-load-images:  k8s-start $(addsuffix -load-images,$(addprefix ${CLUSTER_RULES_PREFIX}-,$(KERNEL_FORWARDER_CONTAINERS)))


.PHONY: k8s-secure-intranet-connectivity-build
k8s-secure-intranet-connectivity-build:

.PHONY: k8s-secure-intranet-connectivity-save
k8s-secure-intranet-connectivity-save:

.PHONY: k8s-secure-intranet-connectivity-load-images
k8s-secure-intranet-connectivity-load-images:

.PHONY: k8s-vppagent-passthrough-nse-build
k8s-vppagent-passthrough-nse-build:

.PHONY: k8s-vppagent-passthrough-nse-save
k8s-vppagent-passthrough-nse-save:

.PHONY: k8s-vppagent-passthrough-nse-load-images
k8s-vppagent-passthrough-nse-load-images:

.PHONY: k8s-skydive-build
k8s-skydive-build:

.PHONY: k8s-skydive-save
k8s-skydive-save: k8s-skydive-build

.PHONY: k8s-skydive-load-images
k8s-skydive-load-images:

.PHONY: k8s-vpn-gateway-nse-build
k8s-vpn-gateway-nse-build: k8s-icmp-responder-nse-build

.PHONY: k8s-vpn-gateway-nse-save
k8s-vpn-gateway-nse-save: k8s-icmp-responder-nse-save

.PHONY: k8s-vpn-gateway-nse-load-images
k8s-vpn-gateway-nse-load-images: k8s-icmp-responder-nse-load-images

.PHONY: k8s-vpn-gateway-nsc-build
k8s-vpn-gateway-nsc-build:

.PHONY: k8s-vpn-gateway-nsc-save
k8s-vpn-gateway-nsc-save:

.PHONY: k8s-vpn-gateway-nsc-load-images
k8s-vpn-gateway-nsc-load-images:

.PHONY: k8s-nsc-build
k8s-nsc-build:

.PHONY: k8s-nsc-save
k8s-nsc-save:

.PHONY: k8s-nsc-load-images
k8s-nsc-load-images:


.PHONY: k8s-nsm-monitor-build
k8s-nsm-monitor-build: ${CONTAINER_BUILD_PREFIX}-nsm-monitor-build

.PHONY: k8s-nsm-monitor-save
k8s-nsm-monitor-save: ${CONTAINER_BUILD_PREFIX}-nsm-monitor-save

.PHONY: k8s-nsm-init-build
k8s-nsm-init-build: ${CONTAINER_BUILD_PREFIX}-nsm-init-build

.PHONY: k8s-nsm-init-save
k8s-nsm-init-save: ${CONTAINER_BUILD_PREFIX}-nsm-init-save

.PHONY: k8s-icmp-responder-nse-build
k8s-icmp-responder-nse-build: ${CONTAINER_BUILD_PREFIX}-test-common-build

.PHONY: k8s-icmp-responder-nse-save
k8s-icmp-responder-nse-save: ${CONTAINER_BUILD_PREFIX}-test-common-save

.PHONY: k8s-icmp-responder-nse-load-images
k8s-icmp-responder-nse-load-images: k8s-test-common-load-images

.PHONY: k8s-vppagent-icmp-responder-nse-build
k8s-vppagent-icmp-responder-nse-build: ${CONTAINER_BUILD_PREFIX}-vpp-test-common-build

.PHONY: k8s-vppagent-icmp-responder-nse-save
k8s-vppagent-icmp-responder-nse-save: ${CONTAINER_BUILD_PREFIX}-vpp-test-common-save

.PHONY: k8s-vppagent-icmp-responder-nse-load-images
k8s-vppagent-icmp-responder-nse-load-images: k8s-vpp-test-common-load-images

.PHONY: k8s-vppagent-firewall-nse-build
k8s-vppagent-firewall-nse-build: ${CONTAINER_BUILD_PREFIX}-vpp-test-common-build

.PHONY: k8s-vppagent-firewall-nse-save
k8s-vppagent-firewall-nse-save: ${CONTAINER_BUILD_PREFIX}-vpp-test-common-save

.PHONY: k8s-vppagent-firewall-nse-load-images
k8s-vppagent-firewall-nse-load-images: k8s-vpp-test-common-load-images

.PHONY: k8s-vppagent-nsc-build
k8s-vppagent-nsc-build: ${CONTAINER_BUILD_PREFIX}-vpp-test-common-build

.PHONY: k8s-vppagent-nsc-save
k8s-vppagent-nsc-save: ${CONTAINER_BUILD_PREFIX}-vpp-test-common-save

.PHONY: k8s-vppagent-nsc-load-images
k8s-vppagent-nsc-load-images: k8s-vpp-test-common-load-images


.PHONY: k8s-crossconnect-monitor-build
k8s-crossconnect-monitor-build: ${CONTAINER_BUILD_PREFIX}-crossconnect-monitor-build

.PHONY: k8s-crossconnect-monitor-save
k8s-crossconnect-monitor-save: ${CONTAINER_BUILD_PREFIX}-crossconnect-monitor-save

.PHONY: k8s-delete-nsm-namespaces
k8s-delete-nsm-namespaces:
	@NSM_NAMESPACE=${NSM_NAMESPACE} ./scripts/delete-nsm-namespaces.sh

.PHONY: k8s-crossconnect-load-images
k8s-crossconnect-monitor-load-images:  k8s-start $(addsuffix -load-images,$(addprefix ${CLUSTER_RULES_PREFIX}-,crossconnect-monitor))

ADMISSION_WEBHOOK_CONTAINERS= admission-webhook
.PHONY: k8s-admission-webhook-build
k8s-admission-webhook-build:  $(addsuffix -build,$(addprefix ${CONTAINER_BUILD_PREFIX}-,$(ADMISSION_WEBHOOK_CONTAINERS)))

.PHONY: k8s-admission-webhook-save
k8s-admission-webhook-save: $(addsuffix -save,$(addprefix ${CONTAINER_BUILD_PREFIX}-,$(ADMISSION_WEBHOOK_CONTAINERS)))

.PHONY: k8s-admission-webhook-load-images
k8s-admission-webhook-load-images:  k8s-start $(addsuffix -load-images,$(addprefix ${CLUSTER_RULES_PREFIX}-,${ADMISSION_WEBHOOK_CONTAINERS}))

.PHONY: k8s-admission-webhook-create-cert
k8s-admission-webhook-create-cert:
	@NSM_NAMESPACE=${NSM_NAMESPACE} ./scripts/webhook-create-cert.sh

.PHONY: k8s-skydive-build
k8s-skydive-build:

.PHONY: k8s-skydive-save
k8s-skydive-save: k8s-skydive-build

.PHONY: k8s-skydive-load-images
k8s-skydive-load-images:

# TODO add k8s-%-logs and k8s-logs to capture all the logs from k8s

.PHONY: k8s-logs
k8s-logs: $(addsuffix -logs,$(addprefix k8s-,$(DEPLOYS)))

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
k8s-check:
	@NSM_NAMESPACE=${NSM_NAMESPACE} ./scripts/nsc_ping_all.sh
	@NSM_NAMESPACE=${NSM_NAMESPACE} ./scripts/verify_vpn_gateway.sh

.PHONY: k8s-logs-snapshot
k8s-logs-snapshot:
	@NSM_NAMESPACE=${NSM_NAMESPACE} ./scripts/logs_snapshot.sh

k8s-logs-snapshot-only-master:
	@NSM_NAMESPACE=${NSM_NAMESPACE} ./scripts/logs_snapshot.sh only-master

.PHONY: k8s-terminating-cleanup
k8s-terminating-cleanup:
	@$(kubectl) get pods -o wide |grep Terminating | cut -d \  -f 1 | xargs $(kubectl) delete pods --force --grace-period 0 {}

.PHONE: k8s-kublet-restart
k8s-kublet-restart: vagrant-kublet-restart

.PHONE: k8s-pods
k8s-pods:
	@$(kubectl) get pods -o wide

.PHONY: k8s-nsmgr-master-tlogs
k8s-nsmgr-master-tlogs:
	@$(kubectl) logs -f $$($(kubectl) get pods -o wide | grep kube-master | grep nsmgr | cut -d\  -f1) -c nsmd

.PHONY: k8s-nsmgr-worker-tlogs
k8s-nsmgr-worker-tlogs:
	@$(kubectl) logs -f $$($(kubectl) get pods -o wide | grep kube-worker | grep nsmgr | cut -d\  -f1) -c nsmd





