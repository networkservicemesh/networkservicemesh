#!/bin/bash

export SC_VERSION="stable" # or "v0.4.7", or "latest"
wget "https://storage.googleapis.com/shellcheck/shellcheck-${SC_VERSION}.linux.x86_64.tar.xz"
tar --xz -xf shellcheck-"${SC_VERSION}".linux.x86_64.tar.xz
sudo cp shellcheck-"${SC_VERSION}"/shellcheck /usr/bin/
shellcheck --version
