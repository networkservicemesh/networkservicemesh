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

set -xe

# Default kubernetes context - if it's "dind" or "minikube" will
# try to bring up a local (dockerized) cluster
# Other kubernetes contexts like "packet", "aws", etc will bring
# up a cluster in the cloud
test -n "${TRAVIS_K8S_CONTEXT}" && set -- "${TRAVIS_K8S_CONTEXT}"

export TEST_CONTEXT=${1:?}

KUBECTL_VERSION=v1.11.0

# Install kubectl
curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/"${KUBECTL_VERSION}"/bin/linux/amd64/kubectl && \
	chmod +x "kubectl" && sudo mv "kubectl" /usr/local/bin/

. scripts/cluster_common.sh
. scripts/integration-tests.sh

# Create the 'minikube' or 'dind' cluster
create_k8s_cluster "${TEST_CONTEXT}"
exit_code=$?
if [ "${exit_code}" != "0" ]
then
    echo "TESTS: FAIL"
    kubectl get pod --namespace=kube-system -lapp=kubernetes-dashboard
    exit "${exit_code}"
fi

# Just exercising some kubectl-s
KUBECTL_BIN=$(command -v kubectl)
kubectl() {
    "${KUBECTL_BIN:?}" --context="${TEST_CONTEXT}" "${@}"
}

# run_tests returns an error on failure
run_tests
exit_code=$?
delete_k8s_cluster "${TEST_CONTEXT}"
if [ "${exit_code}" == "0" ] ; then
    echo "TESTS: PASS"
else
    error_collection
    echo "TESTS: FAIL"
fi

set +x
exit ${exit_code}

# vim: sw=4 ts=4 et si
