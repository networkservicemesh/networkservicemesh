#!/bin/sh

kill_nse() {
    kill -s $1 "${NSE}"
}

sighup () {
    kill_nse "HUP"
}

sigint () {
    kill_nse "INT"
}

sigterm () {
    kill_nse "TERM"
}

sigquit () {
    kill_nse "QUIT"
}

trap sighup SIGHUP
trap sigint SIGINT
trap sigterm SIGTERM
trap sigquit SIGQUIT

"/bin/${NSE_IMAGE}" &

NSE=$!
wait "${NSE}"
wait "${NSE}"