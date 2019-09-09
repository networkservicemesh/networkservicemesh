#!/usr/bin/env bash

start_time="$(date -u +%s)"
export start_time

export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64

export GREEN='\033[0;32m'
export NC='\033[0m' # No Color

GO_VERSION=1.13
GOPROXY=https://proxy.golang.org

mkdir -p "${BIN_DIR}"

function appToSrc {
    case "$1" in
        admission-webhook)      echo "./k8s/cmd/admission-webhook" ;;
        crossconnect-monitor)   echo "./k8s/cmd/crossconnect-monitor" ;;
        nsm-init) echo "./side-cars/cmd/nsm-init" ;;
        nsm-monitor) echo "./side-cars/cmd/nsm-monitor" ;;
        nsmd) echo "./controlplane/cmd/nsmd" ;;
        nsmdp) echo "./k8s/cmd/nsmdp" ;;
        nsmd-k8s) echo "./k8s/cmd/nsmd-k8s" ;;
        kernel-forwarder) echo "./dataplane/kernel-forwarder/cmd/kernel-forwarder.go" ;;
        proxy-nsmd) echo "./controlplane/cmd/proxynsmd" ;;
        proxy-nsmd-k8s) echo "./k8s/cmd/proxy-nsmd-k8s" ;;
        icmp-responder-nse) echo "./test/applications/cmd/icmp-responder-nse" ;;
        monitoring-nsc) echo "./test/applications/cmd/monitoring-nsc" ;;
        proxy-xcon-monitor) echo "./test/applications/cmd/proxy-xcon-monitor" ;;
        monitoring-dns-nsc) echo "./test/applications/cmd/monitoring-dns-nsc" ;;
        vppagent-nsc) echo "./test/applications/cmd/vppagent-nsc" ;;
        vppagent-icmp-responder-nse) echo "./test/applications/cmd/vppagent-icmp-responder-nse" ;;
        vppagent-firewall-nse) echo "./test/applications/cmd/vppagent-firewall-nse" ;;
        vppagent-dataplane) echo "./dataplane/vppagent/cmd" ;;
        nsm-coredns) echo "./k8s/cmd/nsm-coredns" ;;
        *)
        echo "Failed to map app name to source"
        exit 1
    esac
}

function compileApp {
    src=$(appToSrc "$1")
    echo "Compile ${src}"
    if ! go build -i -ldflags "-extldflags '-static' -X  main.version=${VERSION}" -o "${BIN_DIR}/$1" "${src}"
    then
        echo "Failed to compile $1";
        exit $?;
    fi
}

function getElapsed {
    end_time="$(date -u +%s)"
    elapsed=$((end_time-start_time))
    start_time=end_time
    echo ${elapsed}
}