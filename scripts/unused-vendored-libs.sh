#!/bin/sh
DIR=$(pwd)
if [ "$1" != "" ]; then
	DIR=$DIR/$1
fi

cd $DIR

jq -r .package[].path vendor/vendor.json | \
	xargs -I{} sh -c \
	'echo -n "Checking {} ... "; govendor remove {}; make test >/dev/null 2>&1; if [ $? -eq 0 ]; then echo "UNUSED"; else echo "ok"; fi; git reset HEAD --hard >/dev/null 2>&1'
