#!/bin/bash
FLAGS=$1
CUR_DIR=$(dirname "$0")
TEST_LIST=$("$CUR_DIR"/for-each-module.sh "go list ./..." | grep -v -e "sample" | grep -e "^github")
echo $TEST_LIST
# shellcheck disable=SC2086
gotestsum --junitfile ~/junit/unit-tests.xml -- $FLAGS $TEST_LIST
