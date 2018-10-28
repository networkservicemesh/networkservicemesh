#!/bin/bash

docker build -t nsmd/nsmd -f build/Dockerfile.nsmd ..
docker run -it -v "/var/lib/networkservicemesh:/var/lib/networkservicemesh" nsmd/nsmd
docker build -t nsmd/nse -f build/Dockerfile.nse ..
docker run -it -v "/var/lib/networkservicemesh:/var/lib/networkservicemesh" nsmd/nse
docker build -t nsmd/nsc -f build/Dockerfile.nsc ..
docker run -it -v "/var/lib/networkservicemesh:/var/lib/networkservicemesh" nsmd/nsc
docker run --network=host --privileged=true --volume=/var/run:/var/run --volume=/var/lib:/var/lib --volume=/lib/modules:/lib/modules --ipc=host --pid=host -it networkservicemesh/vpp:latest
docker run --network=host --privileged=true --volume=/var/run:/var/run --volume=/var/lib:/var/lib --volume=/lib/modules:/lib/modules --volume=/var/lib/networkservicemesh:/var/lib/networkservicemesh/ --ipc=host --pid=host -it networkservicemesh/vpp-daemon:latest
