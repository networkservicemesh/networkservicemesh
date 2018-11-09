#!/bin/bash

if [ -d "/vagrant/images" ]; then
    for i in /vagrant/images/*; do
        echo "Loading into docker: $i"
        docker load -i "$i"
    done
fi
