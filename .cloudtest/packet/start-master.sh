#!/bin/sh
kubeadm init --pod-network-cidr=192.168.0.0/16 --skip-token-print

echo "KUBELET_EXTRA_ARGS= --read-only-port=10255" > /etc/default/kubelet
service kubelet restart

mkdir -p "$HOME"/.kube
sudo cp -f /etc/kubernetes/admin.conf "$HOME"/.kube/config
sudo chown "$(id -u):$(id -g)" "$HOME"/.kube/config

kubectl apply -f "https://cloud.weave.works/k8s/net?k8s-version=$(kubectl version | base64 | tr -d '\n')&env.IPALLOC_RANGE=192.168.0.0/16"

kubectl taint nodes --all node-role.kubernetes.io/master-

kubeadm token create --print-join-command > join-cluster.sh