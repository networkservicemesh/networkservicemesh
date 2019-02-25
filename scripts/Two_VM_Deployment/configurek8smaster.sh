#!/bin/bash
# Get the IP address that VirtualBox has given this VM
IPADDR=$(ifconfig ens160 | grep -i Mask | awk '{print $2}'| cut -f2 -d:)
echo This VM has IP address "$IPADDR"

#master is the user name of Master node
cd /home/master
# clone networkervicemesh project into VM master
git clone https://github.com/networkservicemesh/networkservicemesh && cd networkservicemesh

# Set up Kubernetes
NODENAME=$(hostname -s)

echo hostname is "$NODENAME"
kubeadm init --apiserver-cert-extra-sans="$IPADDR" --apiserver-advertise-address="$IPADDR" --node-name "$NODENAME" --pod-network-cidr="10.32.0.0/12"
#kubeadm reset --apiserver-cert-extra-sans="$IPADDR" --apiserver-advertise-address="$IPADDR" --node-name "$NODENAME" --pod-network-cidr="10.32.0.0/12"

echo "KUBELET_EXTRA_ARGS= --node-ip=${IPADDR}" > /etc/default/kubelet
service kubelet restart

# Set up admin creds for the master user
echo Copying credentials to /home/master...
sudo --user=master mkdir -p /home/master/.kube
cp -i /etc/kubernetes/admin.conf /home/master/.kube/config
chown "$(id -u master):$(id -g master)" /home/master/.kube/config

# Set up admin creds for the root user
echo Copying credentials to /root
mkdir -p /root/.kube
cp -i /etc/kubernetes/admin.conf /root/.kube/config

# Make credentials available outside of master
echo Copying credentials out of user master
mkdir -p /home/master/networkservicemesh/scripts/Two_VM_Deployment/.kube/
cp /etc/kubernetes/admin.conf /home/master/networkservicemesh/scripts/Two_VM_Deployment/.kube/config

echo "Attempting kubectl version"
kubectl version

# Install networking
kubectl apply -f "https://cloud.weave.works/k8s/net?k8s-version=$(kubectl version | base64 | tr -d '\n')"

# Untaint master
echo "Untainting Master"
kubectl taint nodes --all node-role.kubernetes.io/master-

# Save the kubeadm join command with token
echo '#!/bin/sh' > /home/master/networkservicemesh/scripts/Two_VM_Deployment/kubeadm_join_cmd.sh
kubeadm token create --print-join-command >> /home/master/networkservicemesh/scripts/Two_VM_Deployment/kubeadm_join_cmd.sh
