# Copyright (c) 2018 Cisco and/or its affiliates.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This file includes definitions for Docker images used by the Makefile
# and docker build infrastructure. It also contains the targets to build
# and push Docker images

ORG=networkservicemesh

.PHONY: docker-build-%
docker-build-%:
	@${DOCKERBUILD} -t ${ORG}/$* -f build/Dockerfile.$* ../../
	@if [ "x${COMMIT}" != "x" ] ; then \
		docker tag ${ORG}/$* ${ORG}/$*:${COMMIT} ;\
	fi

#
# Targets to push docker images
#
# NOTE: These assume that ${COMMIT} is set and are meant to be called from travis-ci only.
#
.PHONY: docker-login
docker-login:
	@echo "${DOCKER_PASSWORD}" | docker login -u "${DOCKER_USERNAME}" --password-stdin

.PHONY: docker-push-%
docker-push-%: docker-login
	docker tag ${ORG}/$*:${COMMIT} ${ORG}/$*:${TAG}
	docker tag ${ORG}/$*:${COMMIT} ${ORG}/$*:${BUILD_TAG}
	docker push ${ORG}/$*

#
# Targets to save docker images
#
.PHONY: docker-save-%
docker-save-%:
	docker save -o ../../scripts/vagrant/images/$*.tar ${ORG}/$*
