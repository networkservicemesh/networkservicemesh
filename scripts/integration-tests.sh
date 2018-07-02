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

. scripts/integration-test-helpers.sh

function run_tests() {
    set -xe
    kubectl get nodes
    kubectl version
    kubectl api-versions
    kubectl label --overwrite --all=true nodes app=networkservice-node
    kubectl create -f conf/sample/networkservice-daemonset.yaml
    #
    # Now let's wait for all pods to get into running state
    #
    wait_for_pods default

    # Wait til settles
    echo "INFO: Waiting for Network Service Mesh daemonset to be up and CRDs to be available ..."
    typeset -i cnt=120
    until kubectl get crd | grep networkservicechannels.networkservicemesh.io ; do
        ((cnt=cnt-1)) || error_collection
        sleep 2
    done
    typeset -i cnt=120
    until kubectl get crd | grep networkserviceendpoints.networkservicemesh.io ; do
        ((cnt=cnt-1)) || error_collection
        sleep 2
    done
    typeset -i cnt=120
    until kubectl get crd | grep networkservices.networkservicemesh.io ; do
        ((cnt=cnt-1)) || error_collection
        sleep 2
    done

    #
    # Since daemonset is up and running, create CRD resources
    #
    kubectl create -f conf/sample/networkservice-channel.yaml
    kubectl create -f conf/sample/networkservice-endpoint.yaml
    kubectl create -f conf/sample/networkservice.yaml
    kubectl logs $(kubectl get pods -o name | sed -e 's/.*\///')

    #
    # Starting nsm client pod
    #
    kubectl create -f conf/sample/nsm-client.yaml

    #
    # Now let's wait for nsm-cient pod to get into running state
    #
    wait_for_pods default

    #
    # Final log collection
    #
    kubectl get nodes
    kubectl get pods
    kubectl get crd
    kubectl logs $(kubectl get pods -o name | grep nsm-client ) -c nsm-init
    kubectl get NetworkService,NetworkServiceEndpoint,NetworkServiceChannel --all-namespaces

    # Need to get kubeconfig full path
    # NOTE: Disable this for now until we fix the timing issue
    K8SCONFIG=$HOME/.kube/config
    go test ./plugins/crd/... -v --kube-config=$K8SCONFIG

    # We're all good now
    return 0
}

# vim: sw=4 ts=4 et si
