#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
CONTAINERTXT=/tmp/container.txt
# Prefer docker, but fall back to podman for docker-compatible tooling depending on the container runtime available
RUNTIME="$(shell command -v docker || command -v podman)"

$RUNTIME build -t networkservicemesh/nsmd -f "${DIR}/../build/Dockerfile.nsmd" "${DIR}/../../"
$RUNTIME build -t networkservicemesh/test-common -f "${DIR}/../../test/applications/build/Dockerfile.test-common" "${DIR}/../../"
$RUNTIME build -t networkservicemesh/nsm-init -f "${DIR}/../../side-cars/build/Dockerfile.nsm-init" "${DIR}/../../"
$RUNTIME build -t networkservicemesh/nsm-monitor -f "${DIR}/../../side-cars/build/Dockerfile.nsm-monitor" "${DIR}/../../"

echo "Starting nsmd..."
$RUNTIME run -d -v "/var/lib/networkservicemesh:/var/lib/networkservicemesh" networkservicemesh/nsmd > "${CONTAINERTXT}"
echo "Starting nse..."
$RUNTIME run -d -v "/var/lib/networkservicemesh:/var/lib/networkservicemesh" networkservicemesh/test-common "/bin/icmp-responder-nse" >> "${CONTAINERTXT}"
echo "Starting vpp..."
$RUNTIME run --network=host --privileged=true --volume=/var/run:/var/run --volume=/var/lib:/var/lib --volume=/lib/modules:/lib/modules --ipc=host --pid=host -d networkservicemesh/vpp >> "${CONTAINERTXT}"
echo "Starting vpp-daemon..."
$RUNTIME run --network=host --privileged=true --volume=/var/run:/var/run --volume=/var/lib:/var/lib --volume=/lib/modules:/lib/modules --volume=/var/lib/networkservicemesh:/var/lib/networkservicemesh/ --ipc=host --pid=host -d networkservicemesh/vpp-daemon >> "${CONTAINERTXT}"

echo "vpp-daemon takes time (unnecessarily) to register with nsmd, so let it connect. waiting 60 seconds..."
sleep 60

echo "Running nsm client..."
$RUNTIME run -d -v "/var/lib/networkservicemesh:/var/lib/networkservicemesh" networkservicemesh/nsm-init >> "${CONTAINERTXT}"

echo "Showing nsmd logs..."
$RUNTIME logs "$(sed '1q;d' ${CONTAINERTXT})"

echo "Showing nse logs..."
$RUNTIME logs "$(sed '2q;d' ${CONTAINERTXT})"

echo "Showing vpp-daemon logs..."
$RUNTIME logs "$(sed '4q;d' ${CONTAINERTXT})"

echo "Showing nsc logs..."
$RUNTIME logs "$(sed '5q;d' ${CONTAINERTXT})"

echo "Showing nse interfaces..."
$RUNTIME exec "$(sed '2q;d' ${CONTAINERTXT})" ifconfig -a

echo "Showing nsc interfaces..."
$RUNTIME exec "$(sed '5q;d' ${CONTAINERTXT})" ifconfig -a

echo "Ping nse from nsc interfaces..."
$RUNTIME exec "$(sed '5q;d' ${CONTAINERTXT})" ping -c 5 2.2.2.3

echo "Kill containers..."
xargs "$RUNTIME" kill < ${CONTAINERTXT}
