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

DOCKER_NETMESH_TEST=networkservicemesh/netmesh-test
DOCKER_NETMESH=networkservicemesh/netmesh
DOCKER_SIMPLE_DATAPLANE=networkservicemesh/simple-dataplane
DOCKER_NSM_INIT=networkservicemesh/nsm-init
DOCKER_NSE=networkservicemesh/nse
DOCKER_RELEASE=networkservicemesh/release

#
# Targets to build docker images
#
# NOTE: ${COMMIT} is set in .travis.yml from the first 8 bytes of
# ${TRAVIS_COMMIT}. Thus, for travis-ci builds, we tag the Docker images
# with both the name and this first 8 bytes of the commit hash.
#
.PHONY: docker-build-netmesh-test
docker-build-netmesh-test:
	@${DOCKERBUILD} -t ${DOCKER_NETMESH_TEST} -f build/Dockerfile.nsm-test .
	@if [ "x${COMMIT}" != "x" ] ; then \
		docker tag ${DOCKER_NETMESH_TEST} ${DOCKER_NETMESH_TEST}:${COMMIT} ;\
	fi

.PHONY: docker-build-release
docker-build-release:
	@${DOCKERBUILD} -t ${DOCKER_RELEASE} -f build/Dockerfile .
	@if [ "x${COMMIT}" != "x" ] ; then \
		docker tag ${DOCKER_RELEASE} ${DOCKER_RELEASE}:${COMMIT} ;\
	fi

.PHONY: docker-build-netmesh
docker-build-netmesh: docker-build-release
	@${DOCKERBUILD} -t ${DOCKER_NETMESH} -f build/Dockerfile.nsm .
	@if [ "x${COMMIT}" != "x" ] ; then \
		docker tag ${DOCKER_NETMESH} ${DOCKER_NETMESH}:${COMMIT} ;\
	fi

.PHONY: docker-build-simple-dataplane
docker-build-simple-dataplane: docker-build-release
	@${DOCKERBUILD} -t ${DOCKER_SIMPLE_DATAPLANE} -f build/Dockerfile.simple-dataplane .
	@if [ "x${COMMIT}" != "x" ] ; then \
		docker tag ${DOCKER_SIMPLE_DATAPLANE} ${DOCKER_SIMPLE_DATAPLANE}:${COMMIT} ;\
	fi

.PHONY: docker-build-nsm-init
docker-build-nsm-init: docker-build-release
	@${DOCKERBUILD} -t ${DOCKER_NSM_INIT} -f build/Dockerfile.nsm-init .
	@if [ "x${COMMIT}" != "x" ] ; then \
		docker tag ${DOCKER_NSM_INIT} ${DOCKER_NSM_INIT}:${COMMIT} ;\
	fi

.PHONY: docker-build-nse
docker-build-nse: docker-build-release
	@${DOCKERBUILD} -t ${DOCKER_NSE} -f build/Dockerfile.nse .
	@if [ "x${COMMIT}" != "x" ] ; then \
		docker tag ${DOCKER_NSE} ${DOCKER_NSE}:${COMMIT} ;\
	fi

#
# Targets to push docker images
#
# NOTE: These assume that ${COMMIT} is set and are meant to be called from travis-ci only.
#
.PHONY: docker-login
docker-login:
	@echo "${DOCKER_PASSWORD}" | docker login -u "${DOCKER_USERNAME}" --password-stdin

.PHONY: docker-push-netmesh
docker-push-netmesh: docker-login
	docker tag ${DOCKER_NETMESH}:${COMMIT} ${DOCKER_NETMESH}:${TAG}
	docker tag ${DOCKER_NETMESH}:${COMMIT} ${DOCKER_NETMESH}:travis-${TRAVIS_BUILD_NUMBER}
	docker push ${DOCKER_NETMESH}

.PHONY: docker-push-simple-dataplane
docker-push-simple-dataplane: docker-login
	docker tag ${DOCKER_SIMPLE_DATAPLANE}:${COMMIT} ${DOCKER_SIMPLE_DATAPLANE}:${TAG}
	docker tag ${DOCKER_SIMPLE_DATAPLANE}:${COMMIT} ${DOCKER_SIMPLE_DATAPLANE}:travis-${TRAVIS_BUILD_NUMBER}
	docker push ${DOCKER_SIMPLE_DATAPLANE}

.PHONY: docker-push-nsm-init
docker-push-simple-nsm-init: docker-login
	docker tag ${DOCKER_NSM_INIT}:${COMMIT} ${DOCKER_NSM_INIT}:${TAG}
	docker tag ${DOCKER_NSM_INIT}:${COMMIT} ${DOCKER_NSM_INIT}:travis-${TRAVIS_BUILD_NUMBER}
	docker push ${DOCKER_NSM_INIT}

.PHONY: docker-push-nse
docker-push-simple-nse: docker-login
	docker tag ${DOCKER_NSE}:${COMMIT} ${DOCKER_NSE}:${TAG}
	docker tag ${DOCKER_NSE}:${COMMIT} ${DOCKER_NSE}:travis-${TRAVIS_BUILD_NUMBER}
	docker push ${DOCKER_NSE}
