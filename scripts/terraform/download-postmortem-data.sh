#!/bin/bash -x
# shellcheck disable=SC2086

SSH_OPTS="-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no"
SOURCE_LOC="/var/tmp/nsm-postmortem/"

mkdir -p ~/postmortem

scp ${SSH_OPTS} -r root@"$(terraform output master.public_ip)":"$SOURCE_LOC" ~/postmortem || true
scp ${SSH_OPTS} -r root@"$(terraform output worker1.public_ip)":"$SOURCE_LOC" ~/postmortem || true
