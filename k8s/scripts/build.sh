#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"

docker build -t nsmd/nsmdp -f "${DIR}/../build/Dockerfile.nsmdp" "${DIR}/../../"
docker build -t nsmd/nsmd-k8s -f "${DIR}/../build/Dockerfile.nsmd-k8s" "${DIR}/../../"
