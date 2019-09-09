#!/usr/bin/env bash

function check_git() {
    if ! [[ $(command -v git) ]] ; then 
        echo "git not found."
        exit 1
    fi 
}

function check_deps() {
    check_git
    for file in $(git diff --name-only); do
        echo "${file}"
        if [[ ${file} == *"go.mod" ]] || [[ ${file} == *"go.sum" ]]; then
            echo "ERROR: ${file} has changes after go build..."
            git diff "${file}"
            exit 2
        fi
    done
}

check_deps "$1"
