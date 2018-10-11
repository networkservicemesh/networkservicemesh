#!/bin/bash

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

if [ ! -z "${CIRCLE_BUILD_NUM}" ] ; then
    CLOUD_DEPLOYMENT_NAME=nsm-ci-${CIRCLE_BUILD_NUM}
else
    CLOUD_DEPLOYMENT_NAME=nsm-$$
fi

CLOUD_DEPLOYMENT_DATA=$(pwd)/.data

docker run \
  -v ${CLOUD_DEPLOYMENT_DATA}:/cncf/data \
  --dns 147.75.69.23 --dns 8.8.8.8 \
  -e NAME=${CLOUD_DEPLOYMENT_NAME} \
  -e CLOUD=packet    \
  -e COMMAND=deploy \
  -e BACKEND=file  \
  -e PACKET_AUTH_TOKEN=${PACKET_AUTH_TOKEN} \
  -e TF_VAR_packet_project_id=${PACKET_PROJECT_ID} \
  -ti registry.cncf.ci/cncf/cross-cloud/provisioning:production

export KUBECONFIG=${CLOUD_DEPLOYMENT_DATA}/kubeconfig
kubectl config rename-context ${CLOUD_DEPLOYMENT_NAME} packet

# Adding cross-cloud's nameserver to resolve cluster IP
# On ubuntu /etc/resolv.conf is actually a symlink

sudo cp /etc/resolv.conf /etc/resolv.conf.new
echo "echo \"nameserver 147.75.69.23\" >> /etc/resolv.conf.new" | sudo sh
sudo rm /etc/resolv.conf
sudo mv /etc/resolv.conf.new /etc/resolv.conf 

# Below is temporary workaround of cross-cloud docker production image, which do
# not create RBAC as a part of deployment process. Must be removed once issue #
# resolved

git clone --depth 1 https://github.com/crosscloudci/cross-cloud.git
kubectl create -f ./cross-cloud/rbac/

# End of workaround

