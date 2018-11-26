#!/bin/bash -x
# shellcheck disable=SC2086

SSH_OPTS="-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no"


# Install kubeadm, kubelet and kubectl
scp ${SSH_OPTS} install-kubernetes.sh root@"$(terraform output master.public_ip)":install-kubernetes.sh
scp ${SSH_OPTS} install-kubernetes.sh root@"$(terraform output worker1.public_ip)":install-kubernetes.sh

ssh ${SSH_OPTS} root@"$(terraform output master.public_ip)" ./install-kubernetes.sh &
ssh ${SSH_OPTS} root@"$(terraform output worker1.public_ip)" ./install-kubernetes.sh &
wait

# master: start kubernetes and create join script
# workers: download kubernetes images
scp ${SSH_OPTS} start-master.sh root@"$(terraform output master.public_ip)":start-master.sh
scp ${SSH_OPTS} download-worker-images.sh root@"$(terraform output worker1.public_ip)":download-worker-images.sh

ssh ${SSH_OPTS} root@"$(terraform output master.public_ip)" ./start-master.sh &
ssh ${SSH_OPTS} root@"$(terraform output worker1.public_ip)" ./download-worker-images.sh &
wait

# Download worker join script
scp ${SSH_OPTS} root@"$(terraform output master.public_ip)":join-cluster.sh /tmp/join-cluster.sh
chmod +x /tmp/join-cluster.sh

# Upload and run worker join script
scp ${SSH_OPTS} /tmp/join-cluster.sh root@"$(terraform output worker1.public_ip)":join-cluster.sh
ssh ${SSH_OPTS} root@"$(terraform output worker1.public_ip)" ./join-cluster.sh &

wait