#!/bin/bash

# force the docker version to 18.06

sudo apt-get install -qq -y --force-yes curl apt-transport-https
sudo apt-get purge -qq -y lxc-docker* || true
sudo curl -sSL https://get.docker.com/ | VERSION=18.06 sh
