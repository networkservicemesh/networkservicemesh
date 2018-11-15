#!/usr/bin/env bash
echo ""
echo "********************************************************************************"
echo "          Welcome to NetworkServiceMesh Dev/Debug environment                   "
echo "********************************************************************************"
echo ""
cd /go/src/github.com/ligato/networkservicemesh || exit 101
export PS1='DevEnv: \[\e[0;32m\]\u\[\e[m\] \[\e[1;34m\]\w\[\e[m\] \[\e[1;32m\]\$\[\e[m\] \[\e[1;37m\]'

dep ensure

go install ./vendor/k8s.io/kube-openapi/cmd/openapi-gen
go install ./vendor/k8s.io/code-generator/cmd/deepcopy-gen
echo "Run generators"
go generate ./...

# Print debug prompt

export PATH=$PATH:/go/src/github.com/ligato/networkservicemesh/scripts
debug.sh
bash