#!/bin/sh
apt update
apt install docker.io -y

apt-get update && apt-get install -y apt-transport-https curl
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
cat <<EOF >/etc/apt/sources.list.d/kubernetes.list
deb https://apt.kubernetes.io/ kubernetes-xenial main
EOF
apt-get update
apt-get install -y kubelet kubeadm kubectl
apt-mark hold kubelet kubeadm kubectl

systemctl daemon-reload
systemctl restart kubelet

swapoff --all

kubeadm init --pod-network-cidr=192.168.0.0/16 --skip-token-print

mkdir -p "$HOME"/.kube
sudo cp -f /etc/kubernetes/admin.conf "$HOME"/.kube/config
sudo chown "$(id -u):$(id -g)" "$HOME"/.kube/config

kubectl apply -f https://docs.projectcalico.org/v3.3/getting-started/kubernetes/installation/hosted/rbac-kdd.yaml
kubectl apply -f https://docs.projectcalico.org/v3.3/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calico-networking/1.7/calico.yaml

kubectl taint nodes --all node-role.kubernetes.io/master-


