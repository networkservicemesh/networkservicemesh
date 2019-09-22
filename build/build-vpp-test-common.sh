#!/usr/bin/env bash

APP=$1
ORG=$2
BIN_DIR="$(pwd)/build/dist/${APP}"

# Include function to map apps
# shellcheck disable=SC1091
source ./build/functions.sh

echo -e "${GREEN}----------------------  Building ${OUT} via Cross compile ${APP} ---------------------- ${NC}"

# 1. vppagent-nsc
compileApp vppagent-nsc

# 2. vppagent-icmp-responder-nse
compileApp vppagent-icmp-responder-nse

# 3. vppagent-firewall-nse
compileApp vppagent-firewall-nse

cp dataplane/vppagent/conf/vpp/startup.conf "${BIN_DIR}/vpp.conf"
cp test/applications/vpp-conf/supervisord.conf "${BIN_DIR}/supervisord.conf"
cp test/applications/vpp-conf/run.sh "${BIN_DIR}/vpp-run.sh"

docker build --network="host" -t "${ORG}" -f- "${BIN_DIR}" <<EOF
    FROM ${VPP_AGENT} as runtime
    RUN rm /opt/vpp-agent/dev/etcd.conf; echo "disabled: true" > /opt/vpp-agent/dev/linux-plugin.conf
    RUN mkdir /tmp/vpp/
    COPY "*" "/bin/"
    RUN mv /bin/vpp.conf /etc/vpp/vpp.conf
    RUN mv /bin/supervisord.conf /etc/supervisord/supervisord.conf

    RUN mkdir /tmp/vpp/vppagent-nsc/; echo 'Endpoint: "0.0.0.0:9113"' > /tmp/vpp/vppagent-nsc/grpc.conf
    RUN mkdir /tmp/vpp/vppagent-icmp-responder-nse/; echo 'Endpoint: "0.0.0.0:9112"' > /tmp/vpp/vppagent-icmp-responder-nse/grpc.conf
    RUN mkdir /tmp/vpp/vppagent-firewall-nse/; echo 'Endpoint: "0.0.0.0:9112"' > /tmp/vpp/vppagent-firewall-nse/grpc.conf
EOF

echo -e "${GREEN}----------------------  Build complete. Elapsed $(getElapsed) seconds ----------------------  ${NC}"