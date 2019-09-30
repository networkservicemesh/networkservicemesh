#!/usr/bin/env bash

APP=$1
ORG=$2
BIN_DIR="$(pwd)/build/dist/${APP}/"

# Include function to map apps
# shellcheck disable=SC1091
source ./build/functions.sh

echo -e "${GREEN}----------------------  Building ${OUT} via Cross compile ${APP} ---------------------- ${NC}"

cp pkg/security/scripts/registration.sh "${BIN_DIR}/registration.sh"

docker build --network="host" -t "${ORG}" -f- "${BIN_DIR}" <<EOF
    FROM gcr.io/spiffe-io/spire-server:0.8.0 as builder

    FROM alpine
    RUN apk add dumb-init
    RUN apk add ca-certificates
    RUN mkdir -p /opt/spire/bin
    COPY --from=builder /opt/spire/bin/spire-server /opt/spire/bin/spire-server

    WORKDIR /opt/spire
    COPY "*" "/bin/"
    RUN mv /bin/registration.sh /opt/spire/registration.sh
    RUN chmod +x /opt/spire/registration.sh
    ENTRYPOINT ["/usr/bin/dumb-init", "/opt/spire/registration.sh"]


EOF

echo -e "${GREEN}----------------------  Build complete. Elapsed $(getElapsed) seconds ----------------------  ${NC}"