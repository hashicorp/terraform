#!/bin/bash

project=$1
version=$2
iteration=$3

cd /go/src/github.com/bobtfish/terraform-provider-nsone
go get
go build .
mkdir /dist && cd /dist
ln -s /go/bin bin
fpm -s dir -t deb --name ${project} \
    --iteration ${iteration} --version ${version} \
    --prefix /usr/ \
    ./bin/
rm bin

