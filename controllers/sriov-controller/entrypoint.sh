#!/bin/bash

#
# Currently kubernetes does not provide a mechanism to change
# ulimit values, in case such modification is required, a custom image
# needs to be built with this script as an entry point. Once required
# ulimit modifications are completed, the original kubernetes entry point will
# be called.

# Set memlock limit to unlimited
ulimit -l unlimited
ulimit -n unlimited

# Call original entrypoint script
exec "${@}"