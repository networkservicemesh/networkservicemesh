#!/usr/bin/env bash

cat /etc/lsb-release
uname -a

# propagate environtment variables to the next pipeline steps using Azure's 'logging command'
function propagate() {
    for V in "$@"; do
      val="${!V}"
      echo "##vso[task.setvariable variable=$V;]$val" # export to other steps
      echo "PROPAGATED to next steps/jobs: $V ($val)"
    done
}

GOBIN=${GOPATH}/bin

export PATH=${GOBIN}:${GOROOT}/bin:${PATH}
export GOPROXY=https://proxy.golang.org,direct
propagate PATH GOPROXY

mkdir -p "${GOBIN}"
mkdir -p "${GOPATH}/pkg"

# move repository to standard location under $GOPATH
mkdir -p "${PROJECTPATH}"
shopt -s dotglob nullglob extglob
mv !(gopath) "${PROJECTPATH}"

# A necessity to enable cross-complilation on Linux
sudo chmod -R guo+w "${GOROOT}/pkg"

# Docker images naming/tagging variables
export CIRCLE_BUILD_NUM=${BUILD_BUILDID}
export CIRCLE_WORKFLOW_ID=1  # TODO: seems unused, remove?
export CONTAINER_TAG="azp.${BUILD_SOURCEVERSION:8:8}"
propagate CONTAINER_TAG CIRCLE_BUILD_NUM CIRCLE_WORKFLOW_ID

echo Environment:       ==========================================================
env
echo
echo GO environment:    ==========================================================
go env
echo
go version

