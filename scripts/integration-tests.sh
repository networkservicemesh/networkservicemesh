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
    kubectl create -f conf/sample/nse.yaml
    kubectl create -f conf/sample/test-dataplane.yaml
    wait_for_pods default
 
    #
    # Starting nsm client pod, nsm-client pod should discover gold-network
    # network service along with its endpoint and interface
    kubectl create -f conf/sample/nsm-client.yaml

    #
    # Now let's wait for nsm-cient pod to get into running state
    #
    wait_for_pods default
    exit_ret=$?
    if [ "${exit_ret}" != "0" ] ; then
        return "${exit_ret}"
    fi

    #
    # tests are failing on minikube for adding sidecar containers,  will enable
    # tests once we move the testing to actual Kubernetes cluster.
    ## Refer https://github.com/kubernetes/website/issues/3956#issuecomment-407895766

    # # Side car tests
    # kubectl create -f conf/sidecar-injector/sample-deployment.yaml
    # wait_for_pods default
    # exit_ret=$?
    # if [ "${exit_ret}" != "0" ] ; then
    #     return "${exit_ret}"
    # fi

    # ## Sample test scripts for adding sidecar components in a Kubernetes cluster
    # SIDECAR_CONFIG=conf/sidecar-injector

    # ## Create SSL certificates
    # $SIDECAR_CONFIG/webhook-create-signed-cert.sh --service sidecar-injector-webhook-svc --secret sidecar-injector-webhook-certs --namespace default

    # ## Copy the cert to the webhook configuration YAML file
    # < $SIDECAR_CONFIG/mutatingWebhookConfiguration.yaml $SIDECAR_CONFIG/webhook-patch-ca-bundle.sh >  $SIDECAR_CONFIG/mutatingwebhook-ca-bundle.yaml

    # kubectl label namespace default sidecar-injector=enabled
    # ## Create all the required components
    # kubectl create -f $SIDECAR_CONFIG/configMap.yaml -f $SIDECAR_CONFIG/ServiceAccount.yaml -f $SIDECAR_CONFIG/server-deployment.yaml -f $SIDECAR_CONFIG/mutatingwebhook-ca-bundle.yaml -f $SIDECAR_CONFIG/sidecarInjectorService.yaml
    # wait_for_pods default
    # exit_ret=$?
    # if [ "${exit_ret}" != "0" ] ; then
    #     return "${exit_ret}"
    # fi

    # kubectl delete "$(kubectl get pods -o name | grep sleep)"
    # wait_for_pods default
    # exit_ret=$?
    # if [ "${exit_ret}" != "0" ] ; then
    #     error_collection
    #     return "${exit_ret}"
    # fi

    # pod_count=$(kubectl get pods | grep sleep | grep Running | awk '{print $2}')
    # if [ "${pod_count}" != "2/2" ]; then
    #     error_collection
    #     return 1
    # fi

    # kubectl describe pod "$(kubectl get pods | grep sleep | grep Running | awk '{print $1}')" | grep status=injected
    # exit_ret=$?
    # if [ "${exit_ret}" != "0" ] ; then
    #     error_collection
    #     return "${exit_ret}"
    # fi
    #
    # Let's check number of injected interfaces and if found,
    # check connectivity between nsm-client and nse
    #
    client_pod_name="$(kubectl get pods --all-namespaces | grep nsm-client | awk '{print $2}')"
    client_pod_namespace="$(kubectl get pods --all-namespaces | grep nsm-client | awk '{print $1}')"
    intf_number="$(kubectl exec "$client_pod_name" -n "$client_pod_namespace" -- ifconfig -a | grep -c nse)"
    if [ "$intf_number" -eq 0 ] ; then
        error_collection
        return 1
    fi
    kubectl exec "$client_pod_name" -n "$client_pod_namespace" -- ping 1.1.1.2 -c 5
    #
    # Final log collection
    #
    kubectl get nodes
    kubectl get pods
    kubectl get crd
    kubectl logs "$(kubectl get pods -o name | grep nse)"
    kubectl logs "$(kubectl get pods -o name | grep nsm-client)" -c nsm-init
    DATAPLANES="$(kubectl get pods -o name | grep test-dataplane | cut -d "/" -f 2)"
    for TESTDP in ${DATAPLANES} ; do
        kubectl logs "${TESTDP}"
    done
    kubectl get NetworkService,NetworkServiceEndpoint --all-namespaces

    # Need to get kubeconfig full path
    # NOTE: Disable this for now until we fix the timing issue
    K8SCONFIG="$HOME"/.kube/config
    go test ./plugins/crd/... -v --kube-config="$K8SCONFIG"

    # We're all good now
    return 0
}

# vim: sw=4 ts=4 et si
