#!/usr/bin/env bash

# Set whether or not to use IPv6 enabled Kubernetes deployment
ENABLE_IPV6=${ENABLE_IPV6:-0}  # temporary

# Get the IP address that VirtualBox has given this VM
if [ "$ENABLE_IPV6" -eq 1 ]; then
    echo "Deploying Kubernetes with IPv6..."
    IPADDR=$(ip -6 addr|awk '{print $2}'|grep -P '^(?!fe80)[[:alnum:]]{4}:.*/64'|cut -d '/' -f1)
    sysctl -w net.ipv6.conf.all.forwarding=1
else
    IPADDR=$(ifconfig eth1 | grep -i Mask | awk '{print $2}'| cut -f2 -d:)
fi
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


