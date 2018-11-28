#!/bin/bash

# Install kubernetes
apt-get update && apt-get install -y apt-transport-https
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
cat <<EOF >/etc/apt/sources.list.d/kubernetes.list
deb http://apt.kubernetes.io/ kubernetes-xenial main
EOF

apt-get update
apt-get install -y kubelet kubeadm kubectl
# kubelet requires swap off
swapoff -a
# keep swap off after reboot
sudo sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab
# sed -i '/ExecStart=/a Environment="KUBELET_EXTRA_ARGS=--cgroup-driver=cgroupfs"' /etc/systemd/system/kubelet.service.d/10-kubeadm.conf
sed -i '0,/ExecStart=/s//Environment="KUBELET_EXTRA_ARGS=--cgroup-driver=cgroupfs"\n&/' /etc/systemd/system/kubelet.service.d/10-kubeadm.conf

# Setup Hugepages
#sed -i '9,/KUBELET_EXTRA_ARGS=--cgroup-driver=cgroupfs/KUBELET_EXTRA_ARGS=--cgroup-driver=cgroupfs --feature-gates HugePages=false/'

# kubeproxy seems to needs these to actually work properly...
sysctl net.bridge.bridge-nf-call-iptables=1
{
    echo ip_vs_rr 
    echo ip_vs_wrr
    echo ip_vs_sh 
    echo ip_vs 
} >> /etc/modprobe
modprobe ip_vs_rr
modprobe ip_vs_wrr 
modprobe ip_vs_sh 
modprobe ip_vs
lsmod | grep ip_vs