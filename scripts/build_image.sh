#!/usr/bin/env bash
DOCKER="./build"

while getopts ":o:a:e:b:g:" opt; do
  case $opt in
  o) # org - mandatory argument
    org="$OPTARG"
    ;;
  a) # app - mandatory argument
    app="$OPTARG"
    ;;
  e) # entrypoint - optional argument, empty by default
    entrypoint="$OPTARG"
    ;;
  b) # binpath - mandatory argument
    binpath="$OPTARG"
    ;;
  g) # --build-arg - optional argument for 'docker build'
    docker_args+=" --build-arg $OPTARG"
    ;;
  \?)
    echo "Invalid option: -$OPTARG" >&2
    exit 1
    ;;
  :)
    echo "Option -$OPTARG requires an argument." >&2
    exit 1
    ;;
  esac
done

echo "####################################################################################"
echo "## org        - " "$org"
echo "## app        - " "$app"
echo "## entrypoint - " "$entrypoint"
echo "## binpath    - " "$binpath"
echo "## build-arg  - " "$docker_args"
echo "####################################################################################"

if [ -f "${DOCKER}/Dockerfile.${app}" ]; then
  echo "FILE EXIST" "${app}"
  # shellcheck disable=SC2086
  if ! docker build ${docker_args} --network="host" -t "${org}/${app}" -f "${DOCKER}/Dockerfile.${app}" "${binpath}"; then
    echo Failed to build image for "${app}"
    exit $?
  fi
  exit 0
fi

if [ -n "$entrypoint" ]; then
  echo "Building image with single entrypoint..."
  if ! docker build"${docker_args}" --network="host" -t "${org}/${app}" -f- "${binpath}" <<EOF; then
    FROM alpine as runtime
    COPY "*" "/bin/"
    ENTRYPOINT ["/bin/${entrypoint}"]
EOF
    echo Failed to build image for "${app}"
    exit $?
  fi
  exit 0
fi

echo "Building image with several binaries..."
if ! docker build"${docker_args}" --network="host" -t "${org}/${app}" -f- "${binpath}" <<EOF; then
    FROM alpine as runtime
    COPY "*" "/bin/"
EOF
  echo Failed to build image for "${app}"
  exit $?
fi
