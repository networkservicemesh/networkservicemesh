#!/bin/bash

git log --format=%B -n 1 | head -n 1 | grep -q -v "$1"