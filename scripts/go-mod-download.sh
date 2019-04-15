#!/usr/bin/env sh

limit=10;
attempt=1;

until test "$attempt" -gt "$limit"  || go mod download; do
    attempt=$(( attempt + 1 ));
    rm -rf "$GOPATH"/pkg/mod/cache/vcs/* # wipe out the vcs cache to overwrite corrupted repos
    test "$attempt" -le "$limit" && echo "Trying again, attempt $attempt";
done

test "$attempt" -le "$limit" # ensure correct exit code
