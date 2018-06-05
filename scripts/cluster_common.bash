
# Copyright (c) 2016-2017 Bitnami
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

# Helper functions to create / manage ephemeral minikube or dind cluster
create_k8s_cluster() {
    local ctx=${1:?}
    # Start a k8s cluster (minikube, dind) if not running
    kubectl api-versions --context=${ctx} >& /dev/null || {
        cluster_up=./scripts/cluster-up-${ctx}.sh
        test -f ${cluster_up} || {
            echo "FATAL: bringing up k8s cluster '${ctx}' not supported"
            exit 255
        }
        ${cluster_up}
    }
}
fixup_rbac() {
    local ctx=${1:?}
    # As of ~Sept/2017 both RBAC'd dind and minikube seem to be missing rules to
    # make kube-dns work properly add some (granted) broad ones:
    kubectl --context=${ctx} get clusterrolebinding kube-dns-admin && return 0
    kubectl --context=${ctx} create clusterrolebinding kube-dns-admin --serviceaccount=kube-system:default --clusterrole=cluster-admin
}

# Print ingress IP on stdout
get_ingress_ip() {
    local ctx=${1:?}
    case ${ctx} in
        minikube) minikube ip;;
        dind)     kubectl get no kube-node-1 -ojsonpath='{.status.addresses[0].address}';;
    esac
}
