#!/usr/bin/env bash

APP=$1
ORG=$2
BIN_DIR="./build/dist/${APP}"

# Include function to map apps
# shellcheck disable=SC1091
source ./build/functions.sh

echo -e "${GREEN}----------------------  Building ${OUT} via Cross compile ${APP} ---------------------- ${NC}"

compileApp vppagent-dataplane

cp dataplane/vppagent/conf/vpp/startup.conf "${BIN_DIR}/vpp.conf"
cp dataplane/vppagent/conf/supervisord/supervisord.conf "${BIN_DIR}/supervisord.conf"

docker build --network="host" -t "${ORG}" -f- "${BIN_DIR}" <<EOF
    FROM ${VPP_AGENT} as runtime
    RUN rm /opt/vpp-agent/dev/etcd.conf; echo "disabled: true" > /opt/vpp-agent/dev/linux-plugin.conf
    RUN mkdir /tmp/vpp/
    COPY "*" "/bin/"

    RUN mv /bin/vpp.conf /etc/vpp/vpp.conf
    RUN mv /bin/supervisord.conf /etc/supervisord/supervisord.conf
    RUN rm /opt/vpp-agent/dev/etcd.conf; echo 'Endpoint: "localhost:9111"' > /opt/vpp-agent/dev/grpc.conf
EOF

echo -e "${GREEN}----------------------  Build complete. Elapsed $(getElapsed) seconds ----------------------  ${NC}"