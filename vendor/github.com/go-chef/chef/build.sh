#!/bin/sh

# fixes build/test problems 
if [ ! -d $GOPATH/src/github.com/go-chef/chef ];  then
  mkdir -p $GOPATH/src/github.com/go-chef
  ln -s ./ $GOPATH/src/github.com/go-chef/chef
fi

set -ex

# Grab dependencies for coveralls.io integration
go get -u github.com/axw/gocov/gocov
go get -u github.com/mattn/goveralls
go get -u github.com/ctdk/goiardi/chefcrypto
go get -u github.com/ctdk/goiardi/authentication 
go get -u github.com/davecgh/go-spew/spew
go get -u github.com/smartystreets/goconvey/convey

# go test -coverprofile=coverage dependency
go get -u golang.org/x/tools/cmd/cover

# Overwrite the coverage file
go test -coverprofile=coverage

# Goveralls
go tool cover -func=coverage
