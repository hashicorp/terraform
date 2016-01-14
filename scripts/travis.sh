#!/bin/bash


# Consistent output so travis does not think we're dead during long running
# tests.
export PING_SLEEP=30
bash -c "while true; do echo \$(date) - building ...; sleep $PING_SLEEP; done" &
PING_LOOP_PID=$!

make testacc
TEST_OUTPUT=$?

kill $PING_LOOP_PID
exit $TEST_OUTPUT
