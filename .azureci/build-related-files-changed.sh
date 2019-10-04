#!/bin/bash
set -eo pipefail

FROM_BRANCH="${1}"

PATTERN=".md$|netlify.toml|scripts/netlify.sh|.png$|$0"

echo "No-build file pattern: \"${PATTERN}\""

# shellcheck disable=SC2086
FILES_CHANGED=$(git diff ${FROM_BRANCH} --name-only)
NOT_MATCHING_PATTERN_FILES_CHANGED=$(echo "$FILES_CHANGED" | grep -E -v "${PATTERN}" || true)

if [ -n "${NOT_MATCHING_PATTERN_FILES_CHANGED}" ]; then
  echo "Files not matching the pattern found:"
  echo "${NOT_MATCHING_PATTERN_FILES_CHANGED}"
else
  if [ -n "${FILES_CHANGED}" ]; then
    echo "Only files matching the pattern changed:"
    echo "${FILES_CHANGED}"
  else
    echo "No changed files found."
  fi
  exit 1
fi
