#!/usr/bin/env bash

if [ $# -eq 0 ] ; then
    echo ""
    echo "Please use ./scripts/debug.sh and one of application names nsmd/nsm-init/vppagent/icmp-responder-nse"
    echo ""
    exit
fi

cd /go/src/github.com/networkservicemesh/networkservicemesh/ || exit 101

echo "Starting debug for $1"

go_file=""
port=40000
mkdir -p /bin
output=/bin/debug
if [ "$1" = "nsmd" ]; then
    go_file=./controlplane/cmd/nsmd
    output=/bin/$1
fi

if [ "$1" = "nsmdp" ]; then
    go_file=./k8s/cmd/nsmdp
    output=/bin/$1
fi

if [ "$1" = "nsmd-k8s" ]; then
    go_file=./k8s/cmd/nsmdp-k8s
    output=/bin/$1
fi

if [ "$1" = "nsm-init" ]; then
    go_file=./side-cars/cmd/nsm-init
    output=/bin/$1
fi

if [ "$1" = "icmp-responder-nse" ]; then
    go_file=./test/applications/cmd/nse/icmp-responder-nse
    output=/bin/$1
fi

if [ "$1" = "vppagent-forwarder" ]; then
    go_file=./forwarder/vppagent/cmd/vppagent-forwarder.go
    output=/bin/$1
fi

if [ "$1" = "crossconnect-monitor" ]; then
    go_file=./k8s/cmd/crossconnect_monitor
    output=/bin/$1
fi


# Debug entry point
mkdir -p ./bin
pkill -f "$output"
echo "Compile and start debug of ${go_file} at port ${port}"

# Prepare environment for NSMD
export NSM_SERVER_SOCKET=/var/lib/networkservicemesh/nsm.forwarder-registrar.io.sock
dlv debug --headless --listen=:${port} --api-version=2 --build-flags "-i"  "${go_file}" --output "${output}"