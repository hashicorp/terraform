#!/bin/bash

project=$1
version=$2
iteration=$3

go get github.com/bobtfish/${project}
mkdir /dist && cd /dist
mkdir /tmp/usrbin
ln -s /nail/opt/bin/${project} /tmp/usrbin/${project}
fpm -s dir -t deb --deb-no-default-config-files --name ${project} \
    --iteration ${iteration} --version ${version} \
    /tmp/usrbin/${project}=/usr/bin/ \
    /go/bin/${project}=/nail/opt/bin/

