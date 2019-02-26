#!/bin/bash -x
# shellcheck disable=SC2086

#IP addresses of master and worker VMs should be passed as arugments

IPADDR_Master=$1
IPADDR_Worker=$2
SSH_OPTS="-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no"

echo "IPADDR_Master" $IPADDR_Master

# Install docker
#On the master
scp ${SSH_OPTS} install_docker.sh : root@${IPADDR_Master}:
#On the worker node
scp ${SSH_OPTS} install_docker.sh : root@${IPADDR_Worker}:


#Install on the master
ssh ${SSH_OPTS} root@${IPADDR_Master} ./install_docker.sh &
wait
# on the worker node
ssh ${SSH_OPTS} root@${IPADDR_Worker} ./install_docker.sh &
wait

# Install kubeadm, kubelet and kubectl
scp ${SSH_OPTS} install_kubernetes.sh : root@${IPADDR_Master}:
scp ${SSH_OPTS} install_kubernetes.sh : root@${IPADDR_Worker}:

ssh ${SSH_OPTS} root@${IPADDR_Master} ./install_kubernetes.sh &
wait
#On the worker
ssh ${SSH_OPTS} root@${IPADDR_Worker} ./install_kubernetes.sh &
wait

# transfert of configure scripts
#On the master
scp ${SSH_OPTS} setupk8smaster.sh root@${IPADDR_Master}:

#Configure the master
ssh ${SSH_OPTS} root@${IPADDR_Master} ./setupk8smaster.sh &
wait

#transfert configure script to the worker node
scp ${SSH_OPTS} setupk8sworker.sh root@${IPADDR_Worker}:

#download worker images
scp ${SSH_OPTS} download-worker-images.sh root@${IPADDR_Worker}:

# launch the downolad images on the worker node
ssh ${SSH_OPTS} root@${IPADDR_Worker} ./download-worker-images.sh &
wait

## configure the worker node, to join the master
ssh ${SSH_OPTS} root@${IPADDR_Worker} ./setupk8sworker.sh &

wait
