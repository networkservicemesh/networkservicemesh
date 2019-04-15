#!/usr/bin/env bash
echo ""
echo "********************************************************************************"
echo "          Welcome to NetworkServiceMesh Dev/Debug environment                   "
echo "********************************************************************************"
echo ""
cd "$GOPATH/src/github.com/networkservicemesh/networkservicemesh" || exit 101
export PS1='DevEnv: \[\e[0;32m\]\u\[\e[m\] \[\e[1;34m\]\w\[\e[m\] \[\e[1;32m\]\$\[\e[m\] \[\e[1;37m\]'