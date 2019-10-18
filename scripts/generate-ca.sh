#!/bin/bash

if [ ! -x "$(command -v openssl)" ]; then
    echo "openssl not found"
    exit 1
fi

echo "creating certs in $1"

openssl req -x509 -newkey rsa:4096 -keyout "$1/key.pem" -out "$1/cert.pem" -days 365 -nodes -subj '/CN=localhost'