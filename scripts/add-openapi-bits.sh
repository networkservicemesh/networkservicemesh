#!/bin/bash
#
# Copyright (c) 2018 Cisco and/or its affiliates.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# The purpose of this script is to add the required lines to the auto-generated
# .go files to allow the openapi-gen script to run and include them in the
# OpenAPI output. This allows for 100% automation going from .proto file to .go
# file to CRD OpenAPI validation in Go code.

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[@]}")/..
FILE="${SCRIPT_ROOT}/pkg/nsm/apis/netmesh/netmesh.pb.go"

WHATTOADD="// +k8s:openapi-gen=true"
PATTERN1="type NetworkServiceChannel struct {"
PATTERN2="type NetworkServiceEndpoint struct {"
PATTERN3="type NetworkService struct {"

if [ "${DEBUG}" == "True" ] ; then
	set -xe
fi

SED="sed"
unameOut="$(uname -s)"
case "${unameOut}" in
	Linux*)		SED="sed";;
	Darwin*)	SED="gsed";;
	*)		SED="sed";;
esac

# Make sure on Darwin gsed is installed
if [ "${unameOut}" == "Darwin" ]; then
	if [ "$(command -v "${SED}")" == "" ] ; then
		echo "Please install gnu-sed using homebrew to ensure you have the gsed command"
		exit 1
	fi
fi

# Add the openapi-gen message
#
# The logic below simply adds ${WHATTOADD} iff it doesn't exist after each
# of the ${PATTERN}'s above.

SEARCH1="$(grep -A 1 "${WHATTOADD}" "${FILE}" || echo "X")"
if [ "${SEARCH1}" != "X" ] ; then
	if [ "$(echo "${SEARCH1}" | grep "${PATTERN1}")" == "" ] ; then
		${SED} -i "/${PATTERN1}/i${WHATTOADD}" "${FILE}"
	fi
else
	${SED} -i "/${PATTERN1}/i${WHATTOADD}" "${FILE}"
fi

SEARCH2="$(grep -A 1 "${WHATTOADD}" "${FILE}" || echo "X")"
if [ "${SEARCH2}" != "X" ] ; then
	if [ "$(echo "${SEARCH2}" | grep "${PATTERN2}")" == "" ] ; then
		${SED} -i "/${PATTERN2}/i${WHATTOADD}" "${FILE}"
	fi
else
	${SED} -i "/${PATTERN2}/i${WHATTOADD}" "${FILE}"
fi

SEARCH3="$(grep -A 1 "${WHATTOADD}" "${FILE}" || echo "X")"
if [ "${SEARCH3}" != "X" ] ; then
	if [ "$(echo "${SEARCH3}" | grep "${PATTERN3}")" == "" ] ; then
		${SED} -i "/${PATTERN3}/i${WHATTOADD}" "${FILE}"
	fi
else
	${SED} -i "/${PATTERN3}/i${WHATTOADD}" "${FILE}"
fi
