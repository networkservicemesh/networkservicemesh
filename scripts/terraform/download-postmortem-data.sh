#!/bin/bash -x
# shellcheck disable=SC2086

cluster_id=${PACKET_CLUSTER_ID:-1}
master_ip="$(terraform output master${cluster_id}.public_ip)"
worker_ip="$(terraform output worker${cluster_id}_1.public_ip)"

SSH_OPTS="-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o IdentitiesOnly=yes"
SOURCE_LOC="/var/tmp/nsm-postmortem/"

mkdir -p ~/postmortem

scp ${SSH_OPTS} -r root@${master_ip}:"$SOURCE_LOC" ~/postmortem${cluster_id} || true
scp ${SSH_OPTS} -r root@${worker_ip}:"$SOURCE_LOC" ~/postmortem${cluster_id} || true
