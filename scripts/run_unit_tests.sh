#!/bin/bash
CUR_DIR=$(dirname "$0")
TEST_LIST=$("$CUR_DIR"/for-each-module.sh "go list ./..." | grep -v -e "sample" | grep -e "^github")

: "${JUNITDIR=~/junit}"
mkdir -p "$JUNITDIR"

# shellcheck disable=SC2086 # disable warning for $TEST_LIST word expansion
gotestsum --junitfile "$JUNITDIR/unit-tests.xml" -- -short $TEST_LIST
