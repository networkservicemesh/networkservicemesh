#!/bin/bash

if echo "${CIRCLE_BRANCH}" | grep -q -v "^pull/" ; then echo Skip check on non pull branch: "${CIRCLE_BRANCH}";exit 0;fi

# Update PR refs
FETCH_REFS="${FETCH_REFS} +refs/pull/${CIRCLE_PR_NUMBER}/head:pr/${CIRCLE_PR_NUMBER}/head"

# Retrieve the refs
# shellcheck disable=SC2086
git fetch -u origin ${FETCH_REFS}

# Set upstream of the branch to refs
git branch --set-upstream-to "pr/${CIRCLE_PR_NUMBER}/head"

# Check status
git status -uno | grep -q "Your branch is up.to.date"