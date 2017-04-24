#!/bin/bash

set -o errexit -o nounset

export KEY=$(cat /dev/urandom | tr -cd 'a-z' | head -c 12)
export PASSWORD=$KEY$(cat /dev/urandom | tr -cd 'A-Z' | head -c 2)$(cat /dev/urandom | tr -cd '0-9' | head -c 2)
