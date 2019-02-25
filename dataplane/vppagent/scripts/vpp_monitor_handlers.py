import os


def should_save_core_dump():
    return os.environ.get('SAVE_COREDUMP') == 'true'


def core_dump_location():
    return os.environ.get('POSTMORTEM_DATA_LOCATION', '/var/tmp/nsm-postmortem/vpp-dataplane')


def save_stacktrace():
    gdb.execute("cd /tmp")
    gdb.execute("set logging file vpp_backtrace")
    gdb.execute("set logging on")
    gdb.execute("thread apply all bt")
    gdb.execute("set logging off")


def save_core_dump():
    gdb.execute("cd " + core_dump_location())
    gdb.execute("generate-core-file")


def stop_handler(event):
    if isinstance(event, gdb.SignalEvent):
        if event.stop_signal in ["SIGSEGV"]:
            save_stacktrace()
            if should_save_core_dump():
                save_core_dump()
            gdb.execute("quit")
        else:
            gdb.execute("cont")


def exit_handler(_):
    gdb.execute("quit")


gdb.events.stop.connect(stop_handler)
gdb.events.exited.connect(exit_handler)
