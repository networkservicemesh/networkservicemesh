#!/usr/bin/env sh

register() {
    /opt/spire/bin/spire-server entry create \
        -data /opt/spire/registration.json \
        -registrationUDSPath /tmp/spire-registration.sock
#         | grep -vqE 'connection refused|no such file or directory'
}

/opt/spire/bin/spire-server run -config /run/spire/config/server.conf -registrationUDSPath /tmp/spire-registration.sock &

echo "Cleanup entries"
rm -rf /run/spire/data/*

echo "Registration entries from registration.json"

sleep 2
/opt/spire/bin/spire-server entry show

while ! register; do
    echo "One more try"
	sleep 1
done

/opt/spire/bin/spire-server entry show

wait