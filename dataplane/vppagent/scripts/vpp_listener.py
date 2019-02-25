#!/usr/bin/env python

import os
import re
import shutil
import signal
import sys
import time
import traceback

POSTMORTEM_DATA_LOCATION = os.environ.get('POSTMORTEM_DATA_LOCATION', '/var/tmp/nsm-postmortem/vpp-dataplane')
COLLECT_POSTMORTEM_DATA = os.environ.get('COLLECT_POSTMORTEM_DATA')


def write_stdout(msg):
    # only eventlistener protocol messages may be sent to stdout
    sys.stdout.write(msg)
    sys.stdout.flush()


def write_stderr(msg):
    sys.stderr.write(msg)
    sys.stderr.flush()


def acknowledge():
    write_stdout('RESULT 2\nOK')


def ready():
    # transition from ACKNOWLEDGED to READY
    write_stdout('READY\n')


def parse_data(data):
    # transition from READY to ACKNOWLEDGED
    return dict([x.split(':') for x in data.split()])


def receive_event():
    # read header line and print it to stderr
    line = sys.stdin.readline()
    write_stderr('EVENT: ' + line)

    # read event payload and print it to stderr
    headers = parse_data(line)
    data = sys.stdin.read(int(headers['len']))
    write_stderr('DATA: ' + data + '\n')

    # ignore non vpp events, skipping
    parsed_data = parse_data(data)
    return parsed_data


def kill_supervisord():
    try:
        with open('/run/supervisord.pid', 'r') as pidfile:
            pid = int(pidfile.readline())
        write_stderr('Killing supervisord with pid: ' + str(pid) + '\n')
        os.kill(pid, signal.SIGQUIT)
    except Exception as e:
        write_stderr('Could not kill supervisor: ' + str(e) + '\n')


def collect(src_dir, pattern, dst_dir, timestamp):
    try:
        if not os.path.exists(dst_dir):
            os.makedirs(dst_dir, exist_ok=True)

        matcher = re.compile(pattern)
        matched = [os.path.join(src_dir, filename) for filename in os.listdir(src_dir) if matcher.match(filename)]
        matched_files = [src_path for src_path in matched if os.path.isfile(src_path)]

        def destination_path(path):
            basename = os.path.basename(path)
            return os.path.join(dst_dir, "%d.%s" % (timestamp, basename))

        for src_path in matched_files:
            dst_path = destination_path(src_path)
            write_stderr("Moving '%s' to '%s'" % (src_path, dst_path))
            shutil.move(src_path, dst_path)

    except (OSError, re.error):
        traceback.print_exc()


def collect_postmortem_data():
    if COLLECT_POSTMORTEM_DATA != 'true':
        write_stderr("Postmortem data collection is disabled (set COLLECT_POSTMORTEM_DATA=true to enable it)")
        return
    timestamp = int(time.time())
    write_stderr("Collecting postmortem data...")
    collect('/tmp', 'agent-stdout', POSTMORTEM_DATA_LOCATION, timestamp)
    collect('/tmp', 'vpp-stdout', POSTMORTEM_DATA_LOCATION, timestamp)
    collect('/tmp', 'vppagent-dataplane-stdout', POSTMORTEM_DATA_LOCATION, timestamp)
    collect('/tmp', 'api_post_mortem', POSTMORTEM_DATA_LOCATION, timestamp)
    collect('/tmp', 'vpp_backtrace', POSTMORTEM_DATA_LOCATION, timestamp)
    collect('/var/log/vpp', 'vpp.log', POSTMORTEM_DATA_LOCATION, timestamp)
    collect('/var/log', 'supervisord.log', POSTMORTEM_DATA_LOCATION, timestamp)
    collect('/var/log', 'syslog', POSTMORTEM_DATA_LOCATION, timestamp)
    collect(POSTMORTEM_DATA_LOCATION, 'core', POSTMORTEM_DATA_LOCATION, timestamp)
    write_stderr("Postmortem data collection finished.")


def handle_vpp_exit(event_data):
    collect_postmortem_data()
    kill_supervisord()


# event loop

def main():
    while True:
        ready()

        event_data = receive_event()

        # ignore unwanted processes
        if event_data["processname"] not in ["vpp", "agent"]:
            write_stderr('Ignoring event from ' + event_data["processname"] + '\n')
            acknowledge()
            continue

        # ignore exits with expected exit codes
        if event_data["expected"] == "1":
            write_stderr('Exit state from ' + event_data["processname"] + ' was expected\n')
            acknowledge()
            continue

        handle_vpp_exit(event_data)
        acknowledge()
        return


if __name__ == '__main__':
    main()
