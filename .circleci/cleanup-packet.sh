#!/usr/bin/env bash

pushd ./scripts/terraform && terraform init && popd
.circleci/destroy-cluster.sh