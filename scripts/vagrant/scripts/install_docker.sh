#!/bin/bash

# Copy the vagrant docker provisioner, but force the docker version to 18.06
# see:
# - https://github.com/hashicorp/vagrant/blob/master/plugins/provisioners/docker/cap/debian/docker_install.rb

apt-get install -qq -y --force-yes curl apt-transport-https
apt-get purge -qq -y lxc-docker* || true
curl -sSL https://get.docker.com/ | VERSION=18.06 sh
