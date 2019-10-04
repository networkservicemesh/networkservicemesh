#!/usr/bin/env bash
  #! go fmt ./... 2>&1 #| read

make install-formatter
make format

git diff --exit-code && exit

echo 'Seems like your Go files are not properly formatted. Run "make format" in your branch and commit the changes.'
exit 1
