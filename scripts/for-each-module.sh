#!/bin/bash

res=0
for p in $(find . -name "go.mod") 
do
    d=`dirname $p`
    pushd $d
    $1
    if [ "$?" -ne "0" ] ; then
        res=1
    fi
    popd
done
exit $res
