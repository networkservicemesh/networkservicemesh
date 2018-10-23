#!/bin/bash

for i in /vagrant/images/*; do
    echo "Loading into docker: $i"
    docker load -i "$i"
done