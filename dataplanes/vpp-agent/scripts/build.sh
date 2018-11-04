#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"

docker build -t networkservicemesh/vpp-agent -f "${DIR}/../build/Dockerfile.vpp-agent" "${DIR}/../../../"
docker build -t networkservicemesh/vpp-dataplane -f "${DIR}/../build/Dockerfile.vpp-dataplane" "${DIR}/../../../"
