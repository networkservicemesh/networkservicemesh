#!/usr/bin/env bash

if [ $# -eq 0 ] ; then
    echo ""
    echo "Please use ./scripts/run.sh and one of application names nsmd/nsm-init/vppagent/icmp-responder-nse to compile and run application inside dev container."
    echo ""
    exit
fi

cd /go/src/github.com/networkservicemesh/networkservicemesh/ || exit 101

go_file=""
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
    go_file=./k8s/cmd/nsmd-k8s
    output=/bin/$1
fi

if [ "$1" = "nsm-init" ]; then
    go_file=./side-cars/cmd/nsm-init/init.go
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
echo "Compile and start of ${go_file}"

# Prepare environment for NSMD
export NSM_SERVER_SOCKET=/var/lib/networkservicemesh/nsm.forwarder-registrar.io.sock
CGO_ENABLED=0 GOOS=linux go build -i -ldflags '-extldflags "-static"' -o "${output}" "${go_file}"
"${output}"