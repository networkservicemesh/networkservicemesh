#!/bin/bash
set -eo pipefail
BRANCH="${1}"
PATTERN=".md$|.circleci/skip_job_on_pattern.sh"

if $(echo "${CIRCLE_BRANCH}" | grep -q -v "^pull/") ; then echo Do not skip on non pull branch: ""${CIRCLE_BRANCH}"";exit 0;fi
FILES_CHANGED=$(git diff "${BRANCH}"... --name-only)
NOT_MATCHING_PATTERN_FILES_CHANGED=$(git diff "${BRANCH}"... --name-only| grep -E -v "${PATTERN}" || true)
if [ -z "${NOT_MATCHING_PATTERN_FILES_CHANGED}" ]; then
  echo "Halting Job because only files matching \"${PATTERN}\" found:"
  echo "FILE_CHANGED: ${FILES_CHANGED}"
  circleci-agent step halt
else
  echo "Running Job because files not matching \"${PATTERN}\" found:"
  echo "NOT_MATCHING_PATTERN_FILES_CHANGED: ${NOT_MATCHING_PATTERN_FILES_CHANGED}"
fi
