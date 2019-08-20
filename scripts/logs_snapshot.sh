#!/bin/bash

kubectl="kubectl -n ${NSM_NAMESPACE}"
path="logs"

if [[ ${STORE_POD_LOGS_IN_FILES} == true ]]; then
  if [[ ${STORE_POD_LOGS_DIR} ]]; then
    path=${STORE_POD_LOGS_DIR}
  fi
  mkdir -p "${path}"/pod
  echo "Created folder ${path}/pod"
fi
for pod in $(${kubectl} -o=name get pods --field-selector status.phase=Running); do
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
for pod in $(${kubectl} -o=name get pods); do
  echo "${pod}"
  if [[ ${STORE_POD_LOGS_IN_FILES} == true ]]; then
    filePath=${path}/${pod}-describe.log
    ${kubectl} describe "${pod}" >> "${filePath}"
    echo "Saved describe for ${pod} in ${filePath}"
  else
    echo "Start describe of ${pod}"
    ${kubectl} describe "${pod}"
    echo "End describe of ${pod}"
  fi
done
