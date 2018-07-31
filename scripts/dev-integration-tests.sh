#!/bin/bash

# Copyright (c) 2016-2017 Bitnami
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

. scripts/integration-tests.sh

set -xe

# Verify the image exists
if [ "x$(docker images|grep networkservicemesh/netmesh)" == "x" ]
then
    echo "Docker image networkservicemesh/netmesh not found"
    echo "Please build the image before running integration tests"
    exit 1
fi
if [ "x$(docker images|grep networkservicemesh/nsm-simple-dataplane)" == "x" ]
then
    echo "Docker image networkservicemesh/nsm-simple-dataplane not found"
    echo "Please build the image before running integration tests"
    exit 1
fi
if [ "x$(docker images|grep networkservicemesh/nsm-init)" == "x" ]
then
    echo "Docker image networkservicemesh/nsm-init not found"
    echo "Please build the image before running integration tests"
    exit 1
fi
if [ "x$(docker images|grep networkservicemesh/nse)" == "x" ]
then
    echo "Docker image networkservicemesh/nse not found"
    echo "Please build the image before running integration tests"
    exit 1
fi

# run_tests returns an error on failure
run_tests
exit_code=$?
if [ "${exit_code}" == "0" ] ; then
    echo "TESTS: PASS"
else
    error_collection
    echo "TESTS: FAIL"
fi

set +x
exit ${exit_code}

# vim: sw=4 ts=4 et si
