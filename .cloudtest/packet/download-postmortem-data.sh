#!/bin/bash -x
# shellcheck disable=SC2086

master_ip=$1
worker_ip=$2
cluster_id=$3
sshkey=$4

SSH_OPTS="-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o IdentitiesOnly=yes -i ${sshkey}"
SOURCE_LOC="/var/tmp/nsm-postmortem/"

mkdir -p ~/postmortem

scp ${SSH_OPTS} -r root@${master_ip}:"$SOURCE_LOC" ~/postmortem${cluster_id} || true
scp ${SSH_OPTS} -r root@${worker_ip}:"$SOURCE_LOC" ~/postmortem${cluster_id} || true
