#!/usr/bin/env bash

# This script attaches the vpp process with the GDB
# and stores core dump file in case of crash

if [[ "$COLLECT_POSTMORTEM_DATA" != "true" ]]; then
    echo "Postmortem data collection is disabled (set COLLECT_POSTMORTEM_DATA=true to enable it)"
    exit 0
fi

# setup postmortem data location
readonly DEFAULT_POSTMORTEM_DATA_LOCATION=/var/tmp/nsm-postmortem/vpp-dataplane
readonly POSTMORTEM_DATA_LOCATION=${POSTMORTEM_DATA_LOCATION:-"$DEFAULT_POSTMORTEM_DATA_LOCATION"}
readonly GDB_COMMANDS_FILE=/usr/bin/vpp_monitor_commands.gdb

# make sure postmortem data location exists
mkdir -p "$POSTMORTEM_DATA_LOCATION"

# prepare gdb config
{
    echo "set confirm off"
    echo "set backtrace limit 200"
    echo "handle SIGINT pass nostop"
    echo "source /usr/bin/vpp_monitor_handlers.py"
    echo "attach $(supervisorctl -c /etc/supervisord/supervisord.conf pid vpp)"
    echo 'echo GDB Monitor attached successfully\n'
    echo "cont"
} > "$GDB_COMMANDS_FILE"

# attach to the vpp with a gdb monitor
# and store a core dump in case of crash
gdb -x "$GDB_COMMANDS_FILE"