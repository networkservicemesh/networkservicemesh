#!/usr/bin/env bash

echo
yamllint -v
yamllint -c .yamllint.yml --strict .
