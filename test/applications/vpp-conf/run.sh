#!/bin/sh

cat "/tmp/vpp/$VPP_APP/grpc.conf" > /opt/vpp-agent/dev/grpc.conf

echo "Starting $VPP_APP"
exec "/bin/$VPP_APP"