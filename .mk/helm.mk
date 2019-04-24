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

include .mk/gke.mk
include .mk/aws.mk
include .mk/k8s.mk

.PHONY: helm-init-nsm
helm-init-nsm:
	@helm install deployments/helm/nsm

.PHONY: helm-icmp
helm-icmp: helm-init-nsm
	@helm install deployments/helm/icmp-responder

.PHONY: helm-vpp-icmp
helm-vpp-icmp: helm-init-nsm
	@helm install deployments/helm/vpp-icmp-responder

.PHONY: helm-vpn
helm-vpn: helm-init-nsm
	@helm install deployments/helm/vpn

.PHONY: helm-nsmd-monitoring
helm-nsmd-monitoring: helm-init-nsm
	@helm install deployments/helm/nsmd-monitoring

.PHONY: helm-gke
helm-gke:
	@source ./.env/gke.env
	gke-start helm-init-nsm gcloud-check
	# not sure if it's more effective to test separately
	helm-icmp helm-vpp-icmp helm-vpn helm-nsmd-monitoring
	gke-delete

.PHONY: helm-aws
helm-aws: make aws-init
	@source ./.env/aws.env
	aws-start k8s-config helm-init-nsm
	helm-icmp helm-vpp-icmp helm-vpn helm-nsmd-monitoring
	aws-delete