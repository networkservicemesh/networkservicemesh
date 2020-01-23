#!/bin/bash -x
# shellcheck disable=SC2086

master_ip=$1
worker_ip=$2
sshkey=$3

SSH_OPTS="-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o IdentitiesOnly=yes -i ${sshkey}"


# Install kubeadm, kubelet and kubectl
scp ${SSH_OPTS} ./.cloudtest/packet/install-kubernetes.sh root@${master_ip}:install-kubernetes.sh || exit 1
scp ${SSH_OPTS} ./.cloudtest/packet/install-kubernetes.sh root@${worker_ip}:install-kubernetes.sh || exit 2

ssh ${SSH_OPTS} root@${master_ip} ./install-kubernetes.sh &
ssh ${SSH_OPTS} root@${worker_ip} ./install-kubernetes.sh &
wait

# master1: start kubernetes and create join script
# workers: download kubernetes images
scp ${SSH_OPTS} ./.cloudtest/packet/start-master.sh root@${master_ip}:start-master.sh || exit 3
scp ${SSH_OPTS} ./.cloudtest/packet/download-worker-images.sh root@${worker_ip}:download-worker-images.sh || exit 4

pids=""
ssh ${SSH_OPTS} root@${master_ip} ./start-master.sh &
pids+=" $!"

ssh ${SSH_OPTS} root@${worker_ip} ./download-worker-images.sh &
pids+=" $!"

for pid in $pids; do
  echo "waiting for PID $pid"
  wait "$pid"
  exitcode=$?
  if [ "$exitcode" != 0 ]; then
    echo "node setup failed: process exited with code $exitcode, aborting.." && exit 9
  fi
done


# Download worker join script
mkdir -p /tmp/${master_ip}
scp ${SSH_OPTS} root@${master_ip}:join-cluster.sh /tmp/${master_ip}/join-cluster.sh || exit 5
chmod +x /tmp/${master_ip}/join-cluster.sh || exit 6

# Upload and run worker join script
scp ${SSH_OPTS} /tmp/${master_ip}/join-cluster.sh root@${worker_ip}:join-cluster.sh || exit 7
ssh ${SSH_OPTS} root@${worker_ip} ./join-cluster.sh &

wait

echo "Save KUBECONFIG to file"
scp ${SSH_OPTS} root@${master_ip}:.kube/config ${KUBECONFIG} || exit 8