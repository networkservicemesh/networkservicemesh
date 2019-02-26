#!/usr/bin/env bash

# The master IP address is passed as argument
IPADDR_Master=$1

IPADDR=$(ifconfig $(netstat -nr | tail -1 | awk '{print $NF}') | grep -i Mask | awk '{print $2}'| cut -f2 -d:)
echo This VM worker has IP address "$IPADDR"


echo Copying credentials out of worker
mkdir -p /home/worker/.kube/
mkdir -p /root/.kube
#copy the config file from the master
scp master@${IPADDR_Master}:/home/master/networkservicemesh/scripts/Two_VM_Deployment/.kube/config : /home/worker/.kube/

echo bring kubeadmin join
scp master@${IPADDR_Master}:/home/master/networkservicemesh/scripts/Two_VM_Deployment/kubeadm_join_cmd.sh : /home/worker/
cp /home/worker/.kube/config /root/.kube/config
chown "$(id -u worker):$(id -g worker)" /home/worker/.kube/config

# Joining K8s
bash /home/worker/kubeadm_join_cmd.sh

echo "KUBELET_EXTRA_ARGS= --node-ip=${IPADDR}" > /etc/default/kubelet
service kubelet restart --ignore-preflight-errors
