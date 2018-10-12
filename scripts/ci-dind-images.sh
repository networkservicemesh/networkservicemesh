#!/bin/bash
#
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
#
# The purpose of this script is to add the required lines to the auto-generated
# .go files to allow the openapi-gen script to run and include them in the
# OpenAPI output. This allows for 100% automation going from .proto file to .go
# file to CRD OpenAPI validation in Go code.
#
# This script is used for the Network Service Mesh travis-ci setup. In this
# setup, we run "Docker in Docker" using kubeadm-dind-cluster. What this means
# in practice is we build the NSM images in the outer Docker first. This script
# will copy the images from the outer Docker daemon into the inner Docker daemon
# on every node which dind-cluster.sh starts.

# Temporary directory to save images
WORKDIRECTORY="${WORKDIRECTORY:-./workdirectory}"

# List of images to save and restore
# NOTE: We do not want to copy over the networkservicemesh/release image, as it's the base
# image and is huge. Also note we're only looking at the latest images. In the CI, by the
# time this runs, we've tagged images already, so this ensures we only spend time copying
# the latest images over.
NSM_IMAGES="$(docker images | grep networkservicemesh | grep latest | cut -d " " -f 1 | grep -v release)"

# Each node to copy to
KUBERNETES_NODES="$(kubectl get nodes | grep kube- | cut -d " " -f 1)"

# Save every image we build
function save_images {
	local CUR_DIR
	local IMG
	CUR_DIR="$(pwd)"

	mkdir -p "${WORKDIRECTORY}"
	cd "${WORKDIRECTORY}" || return

	# Debug
	set -xe
	docker images | grep networkservicemesh | cut -d " " -f 1 | grep -v release
	set +xe

	for image in $NSM_IMAGES ; do
		echo "Saving docker image ${image}"
		IMG="$(echo "${image}" | cut -d "/" -f 2)"
		docker save "${image}" > "${IMG}.tar"
	done

	cd "${CUR_DIR}" || return
}

# Restore images
function restore_images {
	local CUR_DIR
	local IMG
	CUR_DIR="$(pwd)"

	cd "${WORKDIRECTORY}" || return

	for node in $KUBERNETES_NODES ; do
		for image in $NSM_IMAGES ; do
			IMG="$(echo "${image}" | cut -d "/" -f 2)"
			echo "Copying ${image} to $node:/"
			docker cp "${IMG}".tar "$node":/
			echo "Loading /${image}.tar into $node"
			docker exec "$node" docker load -i /"${IMG}".tar
		done
	done

	cd "${CUR_DIR}" || return
	rm -rf "${WORKDIRECTORY}"
}

# Save all images locally first
save_images

# Now restore the images into each Kubernetes node
restore_images
