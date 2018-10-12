#!/bin/bash

# Copyright (c) 2016-2017 Bitnami
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

# Bring up kubeadm-dind-cluster (docker-in-docker k8s cluster)
DIND_CLUSTER_SH=dind-cluster-v1.11.sh
DIND_URL=https://cdn.rawgit.com/Mirantis/kubeadm-dind-cluster/master/fixed/${DIND_CLUSTER_SH}

# The number of nodes to test with. For now, lets use a single node cluster
export NUM_NODES=1

# Enable RBAC on the API server
export APISERVER_authorization_mode=RBAC

rm -f ${DIND_CLUSTER_SH}
wget ${DIND_URL}
chmod +x ${DIND_CLUSTER_SH}
./${DIND_CLUSTER_SH} up

export PATH="${HOME}/.kubeadm-dind-cluster:${PATH}"
# Wait for Kubernetes to be up and ready
JSONPATH='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}'; until kubectl get nodes -o jsonpath="$JSONPATH" 2>&1 | grep -q "Ready=True"; do sleep 1; done

# Load the freshly built Docker images from the outer Docker into the inner Docker
./scripts/ci-dind-images.sh

# vim: sw=4 ts=4 et si
