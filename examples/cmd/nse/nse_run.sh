#!/bin/sh

kill_nse() {
    kill -s "$1" "${NSE}"
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

trap sighup HUP
trap sigint INT
trap sigterm TERM
trap sigquit QUIT

echo Starting NSE: "${NSE_IMAGE}"
"/bin/${NSE_IMAGE}" &

NSE=$!
wait "${NSE}"
wait "${NSE}"