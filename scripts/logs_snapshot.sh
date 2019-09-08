#!/bin/bash

if ! [[ $1 == "" ]] && [[ $1 == "only-master" ]] && ! [[ ${STORE_LOGS_IN_ANY_CASES} == true ]] ; then
  echo "Logs not saved: env(${STORE_LOGS_IN_ANY_CASES}) is not true"
  exit 0
fi

kubectl="kubectl -n ${NSM_NAMESPACE}"
path="logs"

if [[ ${STORE_POD_LOGS_IN_FILES} == true ]]; then
  if [[ ${STORE_POD_LOGS_DIR} ]]; then
    path=${STORE_POD_LOGS_DIR}
  fi
  mkdir -p "${path}"/pod
  echo "Created folder ${path}/pod"
fi

for pod in $(${kubectl} -o=name get pods); do
  echo "${pod}"
  if [[ ${STORE_POD_LOGS_IN_FILES} == true ]]; then
    filePath=${path}/${pod}.log
    ${kubectl} logs --all-containers=true "${pod}" >> "${filePath}"
    echo "Saved logs for ${pod} in ${filePath}"
  else
    echo "Start logs of ${pod}"
    ${kubectl} logs --all-containers=true "${pod}"
    echo "End logs of ${pod}"
  fi
done

archive=${path}/../$(basename "${path}").zip
zip -r "${archive}" "${path}"
rm -rf "${path}"