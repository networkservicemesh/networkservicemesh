#!/usr/bin/env sh

register() {
    echo "Registration entries from registration.json"

    /opt/spire/bin/spire-server entry create -node \
        -spiffeID spiffe://test.com/spire-agent \
        -selector k8s_sat:agent_sa:spire-agent

    /opt/spire/bin/spire-server entry create \
        -data /run/spire/entries/registration.json \
        -registrationUDSPath /tmp/spire-registration.sock
}

cleanup() {
    echo "Cleanup entries"
    rm -rf /run/spire/data/*
}

/opt/spire/bin/spire-server run -config /run/spire/config/server.conf -registrationUDSPath /tmp/spire-registration.sock &

cleanup

sleep 2
/opt/spire/bin/spire-server entry show

while ! register; do
    echo "One more try"
	sleep 1
    cleanup
done

/opt/spire/bin/spire-server entry show

wait
