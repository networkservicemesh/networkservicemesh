#!/usr/bin/env bash

img_name=$1
root=$(pwd)
docker_hash=$(docker images -q networkservicemesh/${img_name});
cd scripts/vagrant;

load_image() {
    node_name=$1
    node_hash=$(vagrant ssh --no-tty ${node_name} -c "docker images -q networkservicemesh/${img_name}")

    if [ ${docker_hash} != ${node_hash} ]; then
        echo "Loading image ${img_name}.tar to ${node_name}";
        vagrant ssh ${node_name} -c "sudo docker load -i /vagrant/images/${img_name}.tar" > /dev/null 2>&1;
    else
        echo "Local docker image ${img_name} hash: ${docker_hash} ${node_name} hash ${node_hash} are same... No need to load"
    fi
}

if [ -e "./images/${img_name}.tar" ]; then
   load_image master &
   load_image worker &
   wait
else
    echo "Cannot load ${img_name}.tar: scripts/vagrant/images/${img_name}.tar does not exist";
    exit 1;
fi