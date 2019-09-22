#!/usr/bin/env bash

APP=$1
ORG=$2
BIN_DIR="$(pwd)/build/dist/${APP}"

# Include function to map apps
# shellcheck disable=SC1091
source ./build/functions.sh

echo -e "${GREEN}----------------------  Building ${APP} via Cross compile ---------------------- ${NC}"

# 1. icmp-responder-nse
compileApp icmp-responder-nse

# 2. monitoring-nsc
compileApp monitoring-nsc

# 3. xcon-monitor
compileApp proxy-xcon-monitor

# 4. monitoring-dns-nsc
compileApp monitoring-dns-nsc

docker build --network="host" -t "${ORG}" -f- "${BIN_DIR}" <<EOF
    FROM alpine as runtime
    COPY "*" "/bin/"
EOF

echo -e "${GREEN}----------------------  Build complete. Elapsed $(getElapsed) seconds ---------------------- ${NC}"