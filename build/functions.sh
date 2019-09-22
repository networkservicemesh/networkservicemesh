#!/usr/bin/env bash

start_time="$(date -u +%s)"
export start_time

export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64

export GREEN='\033[0;32m'
export NC='\033[0m' # No Color

mkdir -p "${BIN_DIR}"

function appToSrc {
    case "$1" in
        admission-webhook)      echo "./cmd/admission-webhook" ;;
        crossconnect-monitor)   echo "./cmd/crossconnect-monitor" ;;
        nsm-init) echo "./cmd/nsm-init" ;;
        nsm-monitor) echo "./cmd/nsm-monitor" ;;
        nsmd) echo "./cmd/nsmd" ;;
        nsmdp) echo "./cmd/nsmdp" ;;
        nsmd-k8s) echo "./cmd/nsmd-k8s" ;;
        kernel-forwarder) echo "./kernel-forwarder/cmd/kernel-forwarder.go" ;;
        proxy-nsmd) echo "./cmd/proxynsmd" ;;
        proxy-nsmd-k8s) echo "./cmd/proxy-nsmd-k8s" ;;
        icmp-responder-nse) echo "./applications/cmd/icmp-responder-nse" ;;
        monitoring-nsc) echo "./applications/cmd/monitoring-nsc" ;;
        proxy-xcon-monitor) echo "./applications/cmd/proxy-xcon-monitor" ;;
        monitoring-dns-nsc) echo "./applications/cmd/monitoring-dns-nsc" ;;
        vppagent-nsc) echo "./applications/cmd/vppagent-nsc" ;;
        vppagent-icmp-responder-nse) echo "./applications/cmd/vppagent-icmp-responder-nse" ;;
        vppagent-firewall-nse) echo "./applications/cmd/vppagent-firewall-nse" ;;
        vppagent-dataplane) echo "./vppagent/cmd" ;;
        nsm-coredns) echo "./cmd/nsm-coredns" ;;
        *)
        echo "Failed to map app name to source"
        exit 1
    esac
}
function srcWdir {
    case "$1" in
        admission-webhook)      echo "./k8s" ;;
        crossconnect-monitor)   echo "./k8s" ;;
        nsm-init) echo "./side-cars" ;;
        nsm-monitor) echo "./side-cars" ;;
        nsmd) echo "./controlplane" ;;
        nsmdp) echo "./k8s" ;;
        nsmd-k8s) echo "./k8s" ;;
        kernel-forwarder) echo "./dataplane" ;;
        proxy-nsmd) echo "./controlplane" ;;
        proxy-nsmd-k8s) echo "./k8s" ;;
        icmp-responder-nse) echo "./test" ;;
        monitoring-nsc) echo "./test" ;;
        proxy-xcon-monitor) echo "./test" ;;
        monitoring-dns-nsc) echo "./test" ;;
        vppagent-nsc) echo "./test" ;;
        vppagent-icmp-responder-nse) echo "./test" ;;
        vppagent-firewall-nse) echo "./test" ;;
        vppagent-dataplane) echo "./dataplane" ;;
        nsm-coredns) echo "./k8s" ;;
        *)
        echo "Failed to map app name to source"
        exit 1
    esac
}

function compileApp {
    src=$(appToSrc "$1")
    echo "Compile ${src}"
    rootDir=$(srcWdir "$1")
    pushd "${rootDir}" || exit 1
    if ! go build -i -ldflags "-extldflags '-static' -X  main.version=${VERSION}" -o "${BIN_DIR}/$1" "${src}"
    then
        echo "Failed to compile $1";
        exit $?;
    fi
    popd || exit 1
}

function getElapsed {
    end_time="$(date -u +%s)"
    elapsed=$((end_time-start_time))
    start_time=end_time
    echo ${elapsed}
}