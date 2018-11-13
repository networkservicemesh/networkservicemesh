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
    COMMIT=${COMMIT:-latest}
    kubectl get nodes
    typeset -i cnt=240
    until kubectl get nodes --no-headers | grep -v NotReady | grep Ready ; do
        kubectl get pods
        kubectl get crds
        ((cnt=cnt-1)) || return 1
        sleep 2
    done
    kubectl version
    kubectl api-versions
    kubectl label --overwrite --all=true nodes app=nsmd-ds
    #kubectl label --overwrite nodes kube-node-1 app=networkservice-node
    #
    # Now let's wait for all pods to get into running state
    #
    wait_for_pods default
    exit_code=$?
    [[ ${exit_code} != 0 ]] && return ${exit_code}

    kubectl apply -f k8s/conf/cluster-role-admin.yaml
    kubectl apply -f k8s/conf/cluster-role-binding.yaml


    if [[ -v $NSM_DEV ]]
    then
      docker pull networkservicemesh/nsmdp:"${COMMIT}"
      docker pull networkservicemesh/nsmd:"${COMMIT}"
      docker pull networkservicemesh/nsmd-k8s:"${COMMIT}"
      docker pull networkservicemesh/icmp-responder-nse:"${COMMIT}"
      docker pull networkservicemesh/nsc:"${COMMIT}"
    fi

    mkdir -p /tmp/nsmconfigs
    rm /tmp/nsmconfigs/* || true
    cp k8s/conf/* /tmp/nsmconfigs

    yq w -i /tmp/nsmconfigs/nsmd.yaml spec.template.spec.containers[0].image networkservicemesh/nsmdp:"${COMMIT}"
    yq w -i /tmp/nsmconfigs/nsmd.yaml spec.template.spec.containers[1].image networkservicemesh/nsmd:"${COMMIT}"
    yq w -i /tmp/nsmconfigs/nsmd.yaml spec.template.spec.containers[2].image networkservicemesh/nsmd-k8s:"${COMMIT}"

    yq w -i /tmp/nsmconfigs/icmp-responder-nse.yaml spec.template.spec.containers[0].image networkservicemesh/icmp-responder-nse:"${COMMIT}"

    yq w -i /tmp/nsmconfigs/nsc.yaml spec.template.spec.containers[0].image networkservicemesh/nsc:"${COMMIT}"

    kubectl apply -f /tmp/nsmconfigs/nsmd.yaml

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


    # start icmp-responder-nse
    kubectl apply -f /tmp/nsmconfigs/icmp-responder-nse.yaml
    wait_for_pods default

    typeset -i cnt=240
    until kubectl get netsvc | grep icmp-responder; do
        ((cnt=cnt-1)) || return 1
        sleep 2
    done

    typeset -i cnt=240
    until kubectl get nse | grep icmp; do
        ((cnt=cnt-1)) || return 1
        sleep 2
    done


    # start nsc
    kubectl apply -f /tmp/nsmconfigs/nsc.yaml
    wait_for_pods default

    #
    # Final log collection
    #
    #kubectl get netsvc,nse --all-namespaces

    # We're all good now
    return 0
}

# vim: sw=4 ts=4 et si
