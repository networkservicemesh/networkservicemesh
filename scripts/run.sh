#!/usr/bin/env bash

if [ $# -eq 0 ] ; then
    echo ""
    echo "Please use ./scripts/run.sh and one of application names nsmd/nsc/vppagent/icmp-responder-nse"
    echo ""
    exit
fi

cd /go/src/github.com/ligato/networkservicemesh/ || exit 101

go_file=""

mkdir -p /bin
if [ "$1" = "nsmd" ]; then
    go_file=./controlplane/cmd/nsmd
    output=/bin/$1
fi

if [ "$1" = "nsc" ]; then
    go_file=./examples/cmd/nsc
    output=/bin/$1
fi

if [ "$1" = "icmp-responder-nse" ]; then
    go_file=./examples/cmd/icmp-responder-nse
    output=/bin/$1
fi

if [ "$1" = "vppagent-dataplane" ]; then
    go_file=./dataplane/vppagent/cmd/vppagent-dataplane.go
    output=/bin/$1
fi

if [ "$1" = "crossconnect-monitor" ]; then
    go_file=./k8s/cmd/crossconnect_monitor
    output=/bin/$1
fi

if [ "$1" = "vppagent-dataplane" ]; then
    go_file=./dataplane/vppagent/cmd
    output=/bin/$1
fi


# Debug entry point
mkdir -p ./bin
pkill -f "$output"
echo "Compile and run of ${go_file}"

# Prepare environment for NSMD
echo "Compile"
CGO_ENABLED=0 GOOS=linux go build -i -ldflags '-extldflags "-static"' -o "${output}" "${go_file}"
echo "Running"
"${output}"
