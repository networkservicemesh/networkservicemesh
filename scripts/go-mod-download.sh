#!/usr/bin/env bash

limit=10;
attempt=1;

until (( attempt > limit )) || go mod download; do
    attempt=$(( attempt + 1 ));
    rm -rf "$GOPATH"/pkg/mod/cache/vcs/* # wipe out the vcs cache to overwrite corrupted repos
    (( attempt <= limit )) && echo "Trying again, attempt $attempt";
done

(( attempt <= limit )) # ensure correct exit code
