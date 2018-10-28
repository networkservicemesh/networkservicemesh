#!/bin/bash

docker build -t nsmd/nsmd -f build/Dockerfile.nsmd ..
docker build -t nsmd/nse -f build/Dockerfile.nse ..
docker build -t nsmd/nsc -f build/Dockerfile.nsc ..

echo "Starting nsmd..."
docker run -d -v "/var/lib/networkservicemesh:/var/lib/networkservicemesh" nsmd/nsmd >containers.txt
echo "Starting nse..."
docker run -d -v "/var/lib/networkservicemesh:/var/lib/networkservicemesh" nsmd/nse >>containers.txt
echo "Starting vpp..."
docker run --network=host --privileged=true --volume=/var/run:/var/run --volume=/var/lib:/var/lib --volume=/lib/modules:/lib/modules --ipc=host --pid=host -d networkservicemesh/vpp >>containers.txt
echo "Starting vpp-daemon..."
docker run --network=host --privileged=true --volume=/var/run:/var/run --volume=/var/lib:/var/lib --volume=/lib/modules:/lib/modules --volume=/var/lib/networkservicemesh:/var/lib/networkservicemesh/ --ipc=host --pid=host -d networkservicemesh/vpp-daemon >>containers.txt

echo "vpp-daemon takes time (unnecessarily) to register with nsmd, so let it connect. waiting 60 seconds..."
sleep 60

echo "Running nsm client..."
docker run -d -v "/var/lib/networkservicemesh:/var/lib/networkservicemesh" nsmd/nsc >>containers.txt

echo "Showing nsmd logs..."
docker logs "$(sed '1q;d' containers.txt)"

echo "Showing nse logs..."
docker logs "$(sed '2q;d' containers.txt)"

echo "Showing vpp-daemon logs..."
docker logs "$(sed '4q;d' containers.txt)"

echo "Showing nsc logs..."
docker logs "$(sed '5q;d' containers.txt)"

echo "Showing nse interfaces..."
docker exec "$(sed '2q;d' containers.txt)" ifconfig -a

echo "Showing nsc interfaces..."
docker exec "$(sed '5q;d' containers.txt)" ifconfig -a

echo "Ping nse from nsc interfaces..."
docker exec "$(sed '5q;d' containers.txt)" ping -c 5 2.2.2.3

echo "Kill containers..."
cat containers.txt | xargs docker kill
