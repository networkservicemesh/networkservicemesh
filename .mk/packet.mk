# Copyright (c) 2018 Cisco and/or its affiliates.
#
# Licensed under the Apache License, Version 2.0 (the License);
# you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at:
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an AS IS BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

SSH_OPTS := -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no

ifeq ($(wildcard ./scripts/terraform/packet.tfvars),) 
	TF_PACKET_VARS = -auto-approve
else 
	TF_PACKET_VARS = -var-file=packet.tfvars -auto-approve
endif 

.ONESHELL:
packet-init:
	@pushd scripts/terraform
	@terraform init
	@popd

.ONESHELL:
.PHONY: packet-start
packet-start:
	@pushd scripts/terraform
	@terraform apply ${TF_PACKET_VARS}
	@./create-kubernetes-cluster.sh
	@popd

.ONESHELL:
.PHONY: packet-restart
packet-restart: packet-stop packet-start


.ONESHELL:
.PHONY: packet-stop
packet-stop:
	@pushd scripts/terraform
	@terraform destroy ${TF_PACKET_VARS}
	@popd

.PHONY: packet-%-load-images
packet-%-load-images: 
	@echo Skipping, local docker images are used

.ONESHELL:
.PHONY: packet-get-kubeconfig
packet-get-kubeconfig:
	@pushd scripts/terraform
	@scp ${SSH_OPTS} root@`terraform output master.public_ip`:.kube/config ../../kubeconfig
	@popd