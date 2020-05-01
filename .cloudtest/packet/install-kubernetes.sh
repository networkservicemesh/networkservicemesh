#!/bin/sh

KUBERNETES_VERSION=1.18.1-00

curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
cat <<EOF >/etc/apt/sources.list.d/kubernetes.list
deb https://apt.kubernetes.io/ kubernetes-xenial main
EOF

apt-get update
apt-get install -y docker.io
apt-get install -qy kubelet="${KUBERNETES_VERSION}" kubectl="${KUBERNETES_VERSION}" kubeadm="${KUBERNETES_VERSION}"

systemctl daemon-reload
systemctl restart kubelet

swapoff --all