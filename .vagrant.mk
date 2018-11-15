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
	@cd scripts/vagrant; vagrant up

.PHONY: vagrant-restart
vagrant-restart:
	@cd scripts/vagrant; vagrant destroy -f;sleep 2;vagrant up

.PHONY: vagrant-suspend
vagrant-suspend:
	@cd scripts/vagrant; vagrant suspend

.PHONY: vagrant-resume
vagrant-resume:
	@cd scripts/vagrant; vagrant resume

.PHONY: vagrant-ssh
vagrant-ssh:
	@cd scripts/vagrant; vagrant ssh

.PHONY: vagrant-%-load-images
vagrant-%-load-images:
	@if [ -e "scripts/vagrant/images/$*.tar" ]; then \
		echo "Loading image $*.tar"; \
		cd scripts/vagrant; vagrant ssh -c "sudo docker load -i /vagrant/images/$*.tar" > /dev/null 2>&1; \
	else \
		echo "Cannot load $*.tar: scripts/vagrant/images/$*.tar does not exist"; \
		exit 1; \
	fi