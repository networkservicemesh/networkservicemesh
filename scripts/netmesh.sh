#!/bin/bash
#
# Copyright (c) 2018 Cisco and/or its affiliates.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at:
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# This script runs the netmesh container and it's required dependencies.
#
# 1. Checks if the etcd container is running, and starts it if not.
# 2. Starts the kubeproxy.
# 3. Starts the netmesh container.
#
# The netmesh container will now run, you'll see it's debug output, etc.
# When you ctrl-c the netmesh container, the script will cleanup as
# follows:
#
# 1. Stops and removes the netmesh container
# 2. Stops kubeproxy
# 3. Stops (but does not delete) the etcd container.
#

#
# By default, debug is disabled. Set DEBUGENABLED=1 to turn this on, which
# will default to a file log. Set FILELOG and SCREENLOG to 0/1 to enable
# or disable file and screen logging.
#
DEBUGENABLED=0
FILELOG=1
SCREENLOG=0
THELOGFILE=/tmp/netmesh.log

ETCDVERSION=v3.3.5
ETCDNAME=etcd-gcr-${ETCDVERSION}
KUBECTLPIDFILE=/tmp/.kubectl_proxy.pid
NETMESHNAME=netmesh
KUBECONF=""
ETCDV3CONF=""
HTTPCONF=""
HOSTIP=""

function debugLog() {
	local LOG=$1
	local DATE=$(date +%Y%m%d-%H:%M:%S)

	if [ "${DEBUGENABLED}" == "1" ]; then
		if [ "${FILELOG}" == "1" ]; then echo "$DATE ==> $LOG" >> ${THELOGFILE} ; fi
		if [ "${SCREENLOG}" == "1" ]; then echo "$DATE ==> $LOG" ; fi
	fi
}

function helpMessage() {
	cat << EOF
Usage: netmesh.sh -k kube.conf -e etcdv3.conf -p http.conf -i IP address

       -e: Full path (including filename) of etcdv3 configuation file
       -i: The IP address to proxy for access to the kubernetes API
       -k: Full path (including filename) of kubernetes admin configuration file
       -p: Full path (including filename) of http configuration file

NOTE: -e, -i, -k, and -p are required arguments to execute the script.

This script runs the netmesh container and it's required dependencies.

1. Checks if the etcd container is running, and starts it if not.
2. Starts the kubeproxy.
3. Starts the netmesh container.

The netmesh container will now run, you'll see it's debug output, etc.
When you ctrl-c the netmesh container, the script will cleanup as
follows:

1. Stops and removes the netmesh container
2. Stops kubeproxy
3. Stops (but does not delete) the etcd container.

EOF
}

function isContainerRunning() {
	local CONTAINER=$1

	debugLog "Checking for container ${CONTAINER}"
	if [ "$(docker ps | grep ${CONTAINER})" == "" ] ; then return 0; else return 1; fi
}

function isContainerRunningOrStopped() {
	local CONTAINER=$1

	debugLog "Checking for container ${CONTAINER}"
	if [ "$(docker ps -a | grep ${CONTAINER})" == "" ] ; then return 0; else return 1; fi
}

#
# This function will check if the ${ETCDNAME} container is running or stopped,
# and ensure it is either started explicitely or restarted if it is simply
# stopped. This ensures it works if you've never pulled the image down the
# first time.
function startEtcd() {
	isContainerRunningOrStopped ${ETCDNAME}
	local ETCDRUNNING=$?
	if [ "${ETCDRUNNING}" == "0" ] ; then
		debugLog "Starting ${ETCDNAME}"
		docker run -d \
			  -p 22379:22379 \
			  -p 22380:22380 \
			  --mount type=bind,source=/tmp/etcd-data.tmp,destination=/etcd-data \
			  --name ${ETCDNAME} \
			  gcr.io/etcd-development/etcd:${ETCDVERSION} \
			  /usr/local/bin/etcd \
			  --name s1 \
			  --data-dir /etcd-data \
			  --listen-client-urls http://0.0.0.0:22379 \
			  --advertise-client-urls http://0.0.0.0:22379 \
			  --listen-peer-urls http://0.0.0.0:22380 \
			  --initial-advertise-peer-urls http://0.0.0.0:22380 \
			  --initial-cluster s1=http://0.0.0.0:22380 \
			  --initial-cluster-token tkn \
			  --initial-cluster-state new
	else
		debugLog "Restarting ${ETCDNAME}"
		docker restart ${ETCDNAME}
	fi
}

function stopEtcd() {
	debugLog "Stopping ${ETCDNAME} container"
	docker stop ${ETCDNAME}
}

function destroyKubeProxy() {
	local PID
	local KUBERUNNING

	debugLog "Stopping kubectl proxy"
	if [ -f "${KUBECTLPIDFILE}" ] ; then
		PID=$(cat $KUBECTLPIDFILE)
		KUBERUNNING=$(ps axuw | grep ${PID} | grep -v grep)
		if [ "$KUBERUNNING" != "" ] ; then
			debugLog "Killing kubectl with PID ${PID}"
			kill ${PID}
		fi
		rm -f ${KUBECTLPIDFILE}
	fi
}

function setupKubeProxy() {
	local PID

	destroyKubeProxy

	debugLog "Starting kubectl proxy"
	kubectl proxy --port=8080 --address=${HOSTIP} &
	PID=$!
	debugLog "kubectal proxy started with PID ${PID}"

	echo "${PID}" > ${KUBECTLPIDFILE}

	return ${PID}
}

function runNetMesh() {
	debugLog "Starting netmesh container"
	docker run -it --name=${NETMESHNAME} \
		-v ${KUBECONF}:/conf/kube.conf \
		-v ${ETCDV3CONF}:/conf/etcdv3.conf \
		-v ${HTTPCONF}:/conf/http.conf \
		ligato/networkservicemesh/nsm
}

function stopNetMesh() {
	debugLog "Stopping netmesh container"
	docker rm ${NETMESHNAME}
}

#
# Process commandline options
#
while getopts ":hk:e:p:i:" opt; do
	case ${opt} in
		h )
			helpMessage
			exit 0
			;;
		k )
			KUBECONF=${OPTARG}
			;;
		e )
			ETCDV3CONF=${OPTARG}
			;;
		p )
			HTTPCONF=${OPTARG}
			;;
		i )
			HOSTIP=${OPTARG}
			;;
		\? )
			echo "Invalid option: ${OPTARG}" 1>&2
			exit -1
			;;
		: )
			echo "Invalid option: ${OPTARG} requires an argument" 1>&2
			exit -1
			;;
	esac
done
shift $((OPTIND -1))

# Validate the user passed the right arguments
if [ "${KUBECONF}" == "" ] || [ "${ETCDV3CONF}" == "" ] || [ "${HTTPCONF}" == "" ] || [ "${HOSTIP}" == "" ] ; then
	helpMessage
	exit 0
fi

isContainerRunning ${ETCDNAME}
ETCDRUNNING=$?
if [ "${ETCDRUNNING}" == "0" ] ; then
	startEtcd
fi

setupKubeProxy
runNetMesh
stopNetMesh
destroyKubeProxy
stopEtcd
