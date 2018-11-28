#!/usr/bin/env bash

IPADDR=$(ifconfig eth1 | grep -i Mask | awk '{print $2}'| cut -f2 -d:)
echo This VM has IP address "$IPADDR"

echo Copying credentials out of vagrant
mkdir -p /home/vagrant/.kube/
mkdir -p /root/.kube
cp /vagrant/.kube/config /home/vagrant/.kube/config
cp /vagrant/.kube/config /root/.kube/config
chown "$(id -u vagrant):$(id -g vagrant)" /home/vagrant/.kube/config

# Joining K8s
bash /vagrant/scripts/kubeadm_join_cmd.sh

echo "KUBELET_EXTRA_ARGS= --node-ip=${IPADDR}" > /etc/default/kubelet
service kubelet restart


