#!/usr/bin/env bash

if [ $# -eq 0 ] ; then
    echo ""
    echo "Please use ./scripts/debug.sh and one of application names nsmd/nsc/vppagent/icmp-responder-nse"
    echo ""
    exit
fi

cd /go/src/github.com/ligato/networkservicemesh/ || exit 101

go_file=""
port=40000
mkdir -p /bin
output=/bin/debug
if [ "$1" = "nsmd" ]; then
    go_file=./controlplane/cmd/nsmd
    port=40000
    output=/bin/$1
fi

if [ "$1" = "nsc" ]; then
    go_file=./examples/cmd/nsc
    port=40001
    output=/bin/$1
fi

if [ "$1" = "icmp-responder-nse" ]; then
    go_file=./examples/cmd/icmp-responder-nse
    port=40002
    output=/bin/$1
fi

if [ "$1" = "vppagent-dataplane" ]; then
    go_file=./dataplane/vppagent/cmd/vppagent-dataplane.go
    port=40003
    output=/bin/$1
fi

if [ "$1" = "crossconnect-monitor" ]; then
    go_file=./k8s/cmd/crossconnect_monitor
    port=40004
    output=/bin/$1
fi


# Debug entry point
mkdir -p ./bin
pkill -f "$output"
echo "Compile and start debug of ${go_file} at port ${port}"

# Prepare environment for NSMD
export NSM_SERVER_SOCKET=/var/lib/networkservicemesh/nsm.dataplane-registrar.io.sock
dlv debug --headless --listen=:${port} --api-version=2 --build-flags "-i"  "${go_file}" --output "${output}"