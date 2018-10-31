#!/bin/bash

# The only argument this script should ever be called with is '--verify-only'

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[@]}")/..
CODEGEN_PKG="${CODEGEN_PKG:-$(cd "${SCRIPT_ROOT}"; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}"

echo "Calling ${CODEGEN_PKG}/generate-groups.sh"
"${CODEGEN_PKG}"/generate-groups.sh all \
  github.com/ligato/networkservicemesh/pkg/client github.com/ligato/networkservicemesh/pkg/apis \
  networkservicemesh.io:v1 \
  --output-base "${GOPATH}/src/" \
  --go-header-file "${SCRIPT_ROOT}/conf/boilerplate.txt"

echo "Calling ${CODEGEN_PKG}/generate-groups.sh"
"${CODEGEN_PKG}"/generate-groups.sh all \
  github.com/ligato/networkservicemesh/k8s/pkg/networkservice github.com/ligato/networkservicemesh/k8s/pkg/apis \
  networkservice:v1 \
  --output-base "${GOPATH}/src/" \
  --go-header-file "${SCRIPT_ROOT}/conf/boilerplate2.txt"

echo "Generating other deepcopy funcs"
"${GOPATH}"/bin/deepcopy-gen \
  --input-dirs ./pkg/nsm/apis/netmesh --input-dirs ./pkg/nsm/apis/common \
  --go-header-file "${SCRIPT_ROOT}/conf/boilerplate.txt" \
  --bounding-dirs ./pkg/nsm/apis/netmesh --bounding-dirs ./pkg/nsm/apis/common \
  -O zz_generated.deepcopy \
  -o "${GOPATH}/src"

echo "Generating NetworkService CRD deepcopy funcs"
"${GOPATH}"/bin/deepcopy-gen \
  --input-dirs ./k8s/pkg/apis/networkservice/v1 --input-dirs ./k8s/pkg/apis/networkservice/v1 \
  --go-header-file "${SCRIPT_ROOT}/conf/boilerplate2.txt" \
  --bounding-dirs ./k8s/pkg/apis/networkservice/v1 --bounding-dirs ./k8s/pkg/apis/networkservice/v1 \
  -O zz_generated.deepcopy \
  -o "${GOPATH}/src"

echo "Generating openapi structures"
"${GOPATH}"/bin/openapi-gen \
  --input-dirs ./pkg/apis/networkservicemesh.io/v1 --input-dirs ./pkg/nsm/apis/netmesh \
  --output-package github.com/ligato/networkservicemesh/pkg/apis/networkservicemesh.io/v1 \
  --go-header-file "${SCRIPT_ROOT}/conf/boilerplate.txt" \
  --report-filename="${SCRIPT_ROOT}"/.api_violations.report
