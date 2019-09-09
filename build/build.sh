#!/usr/bin/env bash

APP=$1
ORG=$2

BIN_DIR="./build/dist/${APP}"

# Include function to map apps
# shellcheck disable=SC1091
source ./build/functions.sh

echo -e  "${GREEN}----------------------  Building ${APP} via Cross compile ---------------------- ${NC}"

start_time="$(date -u +%s)"
compileApp "$APP"

docker build --network="host" -t "${ORG}" -f- "${BIN_DIR}" <<EOF
    FROM alpine as runtime
    COPY "${OUT}" "/bin"
    ENTRYPOINT ["/bin/${OUT}"]
EOF

echo -e  "${GREEN}----------------------  Build complete. Elapsed $(getElapsed) seconds ---------------------- ${NC}"