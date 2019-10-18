#!/usr/bin/env bash

# this script prepares environment to collect post-mortem data

# setup postmortem data location
readonly DEFAULT_POSTMORTEM_DATA_LOCATION=/var/tmp/nsm-postmortem/vpp-forwarder
readonly POSTMORTEM_DATA_LOCATION=${POSTMORTEM_DATA_LOCATION:-"$DEFAULT_POSTMORTEM_DATA_LOCATION"}

mkdir -p "$POSTMORTEM_DATA_LOCATION"
sysctl -w debug.exception-trace=1
sysctl -w kernel.core_pattern="$POSTMORTEM_DATA_LOCATION/core-%e-%t"
ulimit -c unlimited
echo 2 > /proc/sys/fs/suid_dumpable
