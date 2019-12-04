# Copyright (c) 2019 Cisco and/or its affiliates.
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

SSH_PARAMS=-i "scripts/aws/nsm-key-pair$(NSM_AWS_SERVICE_SUFFIX)" -F scripts/aws/scp-config$(NSM_AWS_SERVICE_SUFFIX) -o StrictHostKeyChecking=no -o IdentitiesOnly=yes

.PHONY: aws-init
aws-init:
	@pushd scripts/aws && \
	./aws-init.sh && \
	popd

.PHONY: aws-start
aws-start:
	@pushd scripts/aws && \
	AWS_REGION=us-east-2 go run ./... Create && \
	popd

.PHONY: aws-restart
aws-restart: aws-destroy aws-start

.PHONY: aws-destroy
aws-destroy:
	@pushd scripts/aws && \
	AWS_REGION=us-east-2 go run ./... Delete && \
	popd

.PHONY: aws-get-kubeconfig
aws-get-kubeconfig:
	@pushd scripts/aws && \
	aws eks update-kubeconfig --name nsm --kubeconfig ../../kubeconfig && \
	popd

.PHONY: aws-upload-nsm
aws-upload-nsm:
	@tar czf - . --exclude=".git" | ssh ${SSH_PARAMS} aws-master "\
	rm -rf nsm && \
	mkdir nsm && \
	cd nsm && \
	tar xvzf -"

.PHONY: aws-download-postmortem
aws-download-postmortem:
	@echo "Not implemented yet."

.PHONY: aws-print-kubelet-log
aws-print-kubelet-log:
	@echo "Master node kubelet log:" ; \
	ssh ${SSH_PARAMS} aws-master "journalctl -u kubelet" ; \
	echo "Woker node kubelet log:" ; \
	ssh ${SSH_PARAMS} aws-worker "journalctl -u kubelet"

.PHONY: aws-%-load-images
aws-%-load-images: 
	@if [ -e "$(IMAGE_DIR)/$*.tar" ]; then \
		echo "Loading image $*.tar to master and worker" ; \
		scp ${SSH_PARAMS} $(IMAGE_DIR)/$*.tar aws-master:~/ & \
		scp ${SSH_PARAMS} $(IMAGE_DIR)/$*.tar aws-worker:~/ ; \
		ssh ${SSH_PARAMS} aws-master "sudo docker load -i $*.tar" & \
		ssh ${SSH_PARAMS} aws-worker "sudo docker load -i $*.tar" ; \
	else \
		echo "Cannot load $*.tar: $(IMAGE_DIR)/$*.tar does not exist.  Try running 'make k8s-$*-save'"; \
		exit 1; \
	fi
