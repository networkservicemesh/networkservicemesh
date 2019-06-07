#!/bin/sh

cat "/tmp/vpp/$TEST_APPLICATION/grpc.conf" > /opt/vpp-agent/dev/grpc.conf

echo "Starting $TEST_APPLICATION"
exec "/bin/$TEST_APPLICATION"