#!/bin/bash

# Set whether or not to use IPv6 enabled Kubernetes deployment
ENABLE_IPV6=${ENABLE_IPV6:-0}

# Setup Hugepages
#echo "Copying /vagrant/10-kubeadm.conf to /etc/systemd/system/kubelet.service.d/10-kubeadm.conf"
#cp /vagrant/10-kubeadm.conf /etc/systemd/system/kubelet.service.d/10-kubeadm.conf

# Get the IP address that VirtualBox has given this VM
if [ "$ENABLE_IPV6" -eq 1 ]; then
    echo "Deploying Kubernetes with IPv6..."
    IPADDR=$(ip -6 addr|awk '{print $2}'|grep -P '^(?!fe80)[[:alnum:]]{4}:.*/64'|cut -d '/' -f1)
    POD_CIDR="fd2c:852b:74d1:4965::/64"
    sysctl -w net.ipv6.conf.all.forwarding=1
else
    IPADDR=$(ifconfig eth1 | grep -i Mask | awk '{print $2}'| cut -f2 -d:)
    POD_CIDR="10.32.0.0/12"
fi
echo This VM has IP address "$IPADDR"

# Set up Kubernetes
NODENAME=$(hostname -s)
kubeadm init --apiserver-cert-extra-sans="$IPADDR" --apiserver-advertise-address="$IPADDR" --node-name "$NODENAME" --pod-network-cidr="$POD_CIDR"

echo "KUBELET_EXTRA_ARGS= --node-ip=${IPADDR}" > /etc/default/kubelet
service kubelet restart

# Set up admin creds for the vagrant user
echo Copying credentials to /home/vagrant...
sudo --user=vagrant mkdir -p /home/vagrant/.kube
cp -i /etc/kubernetes/admin.conf /home/vagrant/.kube/config
chown "$(id -u vagrant):$(id -g vagrant)" /home/vagrant/.kube/config

# Set up admin creds for the root user
echo Copying credentials to /root
mkdir -p /root/.kube
cp -i /etc/kubernetes/admin.conf /root/.kube/config

# Make credentials available outside of vagrant
echo Copying credentials out of vagrant
mkdir -p /vagrant/.kube/
cp /etc/kubernetes/admin.conf /vagrant/.kube/config

echo "Attempting kubectl version"
kubectl version

# Install networking
if [ "$ENABLE_IPV6" -eq 1 ]; then
    # Calico CNI
    curl https://docs.projectcalico.org/v3.5/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calico-networking/1.7/calico.yaml -O
    sed -i -e "s?192.168.0.0/16?$POD_CIDR?g" calico.yaml
    kubectl apply -f calico.yaml
else
    # Weave CNI
    kubectl apply -f "https://cloud.weave.works/k8s/net?k8s-version=$(kubectl version | base64 | tr -d '\n')"
fi

# Untaint master
echo "Untainting Master"
kubectl taint nodes --all node-role.kubernetes.io/master-

# Save the kubeadm join command with token
echo '#!/bin/sh' > /vagrant/scripts/kubeadm_join_cmd.sh
kubeadm token create --print-join-command >> /vagrant/scripts/kubeadm_join_cmd.sh
