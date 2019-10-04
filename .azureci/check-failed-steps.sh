#!/usr/bin/env bash

# There are situations when it is profitable from performance point of view
# to run several steps regardless of their result and fail the job only
# if one of the steps has failed. This might help catch more errors per
# single CI run, sparing CI resources and developer's time.

echo "Checking if any of the previous steps failed..."
echo "AGENT_JOBSTATUS value: '$AGENT_JOBSTATUS'"
if [ "$AGENT_JOBSTATUS" == 'SucceededWithIssues' ]; then
  echo 'Failed step found, aborting...'
  exit 1
fi

echo 'No failed steps so far, continue...'
