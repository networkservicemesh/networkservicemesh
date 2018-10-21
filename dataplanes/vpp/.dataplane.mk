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

DOCKER_VPPDATAPLANE=networkservicemesh/vppdataplane
DOCKER_VPP=networkservicemesh/vpp

#
# Targets to build docker images
#
# NOTE: ${COMMIT} is set in .travis.yml from the first 8 bytes of
# ${TRAVIS_COMMIT}. Thus, for travis-ci builds, we tag the Docker images
# with both the name and this first 8 bytes of the commit hash.
#
.PHONY: docker-build-vppdataplane
docker-build-vppdataplane:
	@${DOCKERBUILD} -t ${DOCKER_VPPDATAPLANE} -f build/Dockerfile.build ../..
	@if [ "x${COMMIT}" != "x" ] ; then \
		docker tag ${DOCKER_VPPDATAPLANE} ${DOCKER_VPPDATAPLANE}:${COMMIT} ;\
	fi

.PHONY: docker-build-vpp
docker-build-vpp:
	@${DOCKERBUILD} -t ${DOCKER_VPP} -f build/Dockerfile.vpp ../..
	@if [ "x${COMMIT}" != "x" ] ; then \
		docker tag ${DOCKER_VPP} ${DOCKER_VPP}:${COMMIT} ;\
	fi

#
# Targets to push docker images
#
# NOTE: These assume that ${COMMIT} is set and are meant to be called from travis-ci only.
#
.PHONY: docker-login
docker-login:
	@echo "${DOCKER_PASSWORD}" | docker login -u "${DOCKER_USERNAME}" --password-stdin

.PHONY: docker-push-vpp-dataplane
docker-push-vppdataplane: docker-login
	docker tag ${DOCKER_VPPDATAPLANE}:${COMMIT} ${DOCKER_VPPDATAPLANE}:${TAG}
	docker tag ${DOCKER_VPPDATAPLANE}:${COMMIT} ${DOCKER_VPPDATAPLANE}:travis-${TRAVIS_BUILD_NUMBER}
	docker push ${DOCKER_VPPDATAPLANE}

.PHONY: docker-push-vpp
docker-push-vpp: docker-login
	docker tag ${DOCKER_VPP}:${COMMIT} ${DOCKER_VPP}:${TAG}
	docker tag ${DOCKER_VPP}:${COMMIT} ${DOCKER_VPP}:travis-${TRAVIS_BUILD_NUMBER}
	docker push ${DOCKER_VPP}