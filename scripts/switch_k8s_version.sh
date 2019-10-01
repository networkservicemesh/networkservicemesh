#!/bin/bash

# Original script by Andy Bursavich:
# https://github.com/kubernetes/kubernetes/issues/79384#issuecomment-521493597

set -euo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/../"

VERSION=${1#"v"}
if [ -z "$VERSION" ]; then
    printf "Usage (assuming k8s 1.16.0): \n %s 1.16.0\n" "${0}"
    exit 1
fi

MODS=()
while IFS='' read -r line
do
    MODS+=("$line")
done < <( curl -sS https://raw.githubusercontent.com/kubernetes/kubernetes/v"${VERSION}"/go.mod | sed -n 's|.*k8s.io/\(.*\) => ./staging/src/k8s.io/.*|k8s.io/\1|p')

grep -r --include=go.mod k8s.io . | awk -F: '{print $1}' | sort -u| sed 's/go.mod$//' | while IFS= read -r DIR
do
    pushd "${DIR}"
    for MOD in "${MODS[@]}"; do
        V=$(
            go mod download -json "${MOD}@kubernetes-${VERSION}" |
            sed -n 's|.*"Version": "\(.*\)".*|\1|p'
        )
        go mod edit "-replace=${MOD}=${MOD}@${V}"
    done
    go get "k8s.io/kubernetes@v${VERSION}"
    go mod tidy
    popd
done