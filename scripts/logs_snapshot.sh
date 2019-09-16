#!/bin/bash

if ! [[ $1 == "" ]] && [[ $1 == "only-master" ]] && ! [[ ${STORE_LOGS_IN_ANY_CASES} == true ]] ; then
  echo "Logs not saved: env(${STORE_LOGS_IN_ANY_CASES}) is not true"
  exit 0
fi

kubectl="kubectl -n ${NSM_NAMESPACE}"
tmp="logs"
pathToSave="logs"

if [[ ${STORE_POD_LOGS_IN_FILES} == true ]]; then
  if [[ ${STORE_POD_LOGS_DIR} ]]; then
    pathToSave=${STORE_POD_LOGS_DIR}
  fi
  mkdir -p "${tmp}"/pod
  echo "Created folder ${path}/pod"
fi

for pod in $(${kubectl} -o=name get pods); do
  echo "${pod}"
  if [[ ${STORE_POD_LOGS_IN_FILES} == true ]]; then
    filePath=${tmp}/${pod}.log
    ${kubectl} logs --all-containers=true "${pod}" >> "${filePath}"
    echo "Saved logs for ${pod} in ${filePath}"
    logs="$(${kubectl} logs --all-containers=true -p ${pod})"
    if [[ "${logs}" == "" ]]; then 
      echo "No previous logs for ${pod}"
      continue
    fi
    echo "${logs}" >> "${tmp}/${pod}.previous.log"
    echo "Saved logs for ${pod} in ${tmp}/${pod}.previous.log"
  else
    echo "Start logs of ${pod}"
    ${kubectl} logs --all-containers=true "${pod}"
    echo "End logs of ${pod}"
  fi
done

archive=${pathToSave}.zip
echo ${archive}
zip -r "${archive}" "${tmp}"
rm -rf "${tmp}"