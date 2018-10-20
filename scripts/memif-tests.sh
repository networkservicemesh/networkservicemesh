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
    kubectl get nodes
    kubectl version
    kubectl api-versions
    kubectl label --overwrite --all=true nodes app=networkservice-node
    #kubectl label --overwrite nodes kube-node-1 app=networkservice-node
    kubectl create -f conf/sample/networkservice-daemonset.yaml
    #
    # Now let's wait for all pods to get into running state
    #
    wait_for_pods default
    exit_code=$?
    [[ ${exit_code} != 0 ]] && return ${exit_code}


    # Wait til settles
    echo "INFO: Waiting for Network Service Mesh daemonset to be up and CRDs to be available ..."
    typeset -i cnt=240
    until kubectl get crd | grep networkserviceendpoints.networkservicemesh.io ; do
        ((cnt=cnt-1)) || return 1
        sleep 2
    done
    typeset -i cnt=240
    until kubectl get crd | grep networkservices.networkservicemesh.io ; do
        ((cnt=cnt-1)) || return 1
        sleep 2
    done

    #
    # Since daemonset is up and running, create CRD resources
    #
    kubectl create -f conf/sample/networkservice.yaml
    wait_for_networkservice default

    #
    # Starting nse pod which will advertise an endpoint for gold-network
    # network service
    kubectl create -f conf/sample/nse-memif.yaml
    kubectl create -f conf/sample/test-dataplane.yaml
    wait_for_pods default
 
    #
    # Starting nsm client pod, nsm-client pod should discover gold-network
    # network service along with its endpoint and interface
    kubectl create -f conf/sample/nsc-memif.yaml

    #
    # Now let's wait for nsm-cient pod to get into running state
    #
    wait_for_pods default
    exit_ret=$?
    if [ "${exit_ret}" != "0" ] ; then
        return "${exit_ret}"
    fi
}

run_tests

# vim: sw=4 ts=4 et si