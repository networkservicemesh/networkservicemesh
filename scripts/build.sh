#!/usr/bin/env bash

APP=$1
ROOT_DIR=$2
APP_DIR=$3
BIN_DIR=$4
VERSION=$5

set +x

echo -e "${GREEN}----------------------  Building ${ROOT_DIR}::${APP} via Cross compile ---------------------- ${NC}"
echo "Binaries dir: ${BIN_DIR}"
start_time="$(date -u +%s)"
pushd "${ROOT_DIR}" 1>/dev/null || exit 1
if ! CGO_ENABLED=0 GOOS=linux go build -ldflags "-extldflags '-static' -X  main.version=${VERSION}" -o "${BIN_DIR}/${APP}" "${APP_DIR}"; then
  echo "Failed to compile $1"
  exit $?
fi
#popd || exit 1
end_time="$(date -u +%s)"

echo -e "${GREEN}----------------------  Build complete. Elapsed $((end_time - start_time)) seconds ---------------------- ${NC}"
