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

.PHONY: vagrant-start
vagrant-start:
	@cd scripts/vagrant; vagrant up --no-parallel;

.PHONY: vagrant-destroy
vagrant-destroy:
	@cd scripts/vagrant; vagrant destroy -f

.PHONY: vagrant-restart
vagrant-restart: vagrant-destroy
	@cd scripts/vagrant; sleep 2;vagrant up --no-parallel

.PHONY: vagrant-suspend
vagrant-suspend:
	@cd scripts/vagrant; vagrant suspend

.PHONY: vagrant-resume
vagrant-resume:
	@cd scripts/vagrant; vagrant resume

.PHONY: vagrant-ssh
vagrant-ssh:
	@cd scripts/vagrant; vagrant ssh master

.PHONY: vagrant-ssh-worker%
vagrant-ssh-worker%:
	@cd scripts/vagrant; vagrant ssh worker$*

.PHONY: vagrant-restart-kubelet
vagrant-restart-kubelet:
	@cd scripts/vagrant; vagrant ssh master -c "sudo service kubelet restart"; \
	number=1 ; while [[ $$number -le ${WORKER_COUNT} ]] ; do \
		vagrant ssh worker$$number	-c "sudo service kubelet restart" ; \
		((number++)) ; \
	done

.PHONY: vagrant-%-load-images
vagrant-%-load-images:
	@if [ -e "$(IMAGE_DIR)/$*.tar" ]; then \
		mkdir -p scripts/vagrant/images ; \
		cp "$(IMAGE_DIR)/$*.tar" scripts/vagrant/images/ ; \
		cd scripts/vagrant; \
		echo "Loading image $*.tar to master"; \
		vagrant ssh master -c "sudo docker rmi networkservicemesh/$* -f && docker load -i /vagrant/images/$*.tar" > /dev/null 2>&1; \
		number=1 ; while [[ $$number -le ${WORKER_COUNT} ]] ; do \
			echo "Loading image $*.tar to worker$$number"; \
			vagrant ssh worker$$number -c "sudo docker rmi networkservicemesh/$* -f && docker load -i /vagrant/images/$*.tar" > /dev/null 2>&1; \
			((number++)) ; \
		done; \
	else \
		echo "Cannot load $*.tar: $(IMAGE_DIR)/$*.tar does not exist.  Try running 'make k8s-$*-save'"; \
		exit 1; \
	fi

.PHONY: vagrant-print-kubelet-log
vagrant-print-kubelet-log:
	@echo "Master node kubelet log:"; \
	cd scripts/vagrant; \
	vagrant ssh master -c "journalctl -u kubelet"; \
	number=1 ; while [[ $$number -le ${WORKER_COUNT} ]] ; do \
		echo "Woker$$number node kubelet log:"; \
		vagrant ssh worker$$number -c "journalctl -u kubelet"; \
		((number++)) ; \
	done;

.PHONY: vagrant-config-location
vagrant-config-location:
	@echo "$(CURDIR)/scripts/vagrant/.kube/config"
