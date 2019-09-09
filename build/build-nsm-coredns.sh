#!/usr/bin/env bash

APP=$1
ORG=$2
BIN_DIR="./build/dist/${APP}/"

# Include function to map apps
# shellcheck disable=SC1091
source ./functions.sh

echo -e "${GREEN}----------------------  Building ${OUT} via Cross compile ${APP} ---------------------- ${NC}"

compileApp nsm-coredns

docker build --network="host" -t "${ORG}" -f- "${BIN_DIR}" <<EOF
    FROM golang:alpine as build
    RUN apk update && apk add --no-cache git ca-certificates tzdata && update-ca-certificates

    FROM alpine as runtime
    COPY "*" "/bin/"
    COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
    EXPOSE 53 53/udp
    ENTRYPOINT ["/bin/nsm-coredns"]

EOF

echo -e "${GREEN}----------------------  Build complete. Elapsed $(getElapsed) seconds ----------------------  ${NC}"