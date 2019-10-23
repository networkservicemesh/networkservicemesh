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

echo "Waiting spire-server to launch on 8081..."

while ! nc -z spire-server 8081; do
  sleep 0.1 # wait for 1/10 of the second before check again
done

echo "spire-server launched"

/opt/spire/bin/spire-server entry show

while ! register; do
    echo "One more try"
	sleep 1
    cleanup
done

/opt/spire/bin/spire-server entry show
echo "spire-entries successfully registered"
wait
