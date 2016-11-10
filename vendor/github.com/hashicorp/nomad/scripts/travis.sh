#!/usr/bin/env bash

export PING_SLEEP=30
bash -c "while true; do echo \$(date) - building ...; sleep $PING_SLEEP; done" &
PING_LOOP_PID=$!

trap "kill $PING_LOOP_PID" EXIT HUP INT QUIT TERM

make test
TEST_OUTPUT=$?

kill $PING_LOOP_PID
exit $TEST_OUTPUT
