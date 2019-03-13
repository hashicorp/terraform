# This Dockerfile builds on golang:alpine by building Terraform from source
# using the current working directory.
#
# This produces a docker image that contains a working Terraform binary along
# with all of its source code, which is what gets released on hub.docker.com
# as terraform:full. The main releases (terraform:latest, terraform:light and
# the release tags) are lighter images including only the officially-released
# binary from releases.hashicorp.com; these are built instead from
# scripts/docker-release/Dockerfile-release.

FROM golang:alpine
LABEL maintainer="HashiCorp Terraform Team <terraform@hashicorp.com>"

RUN apk add --update git bash openssh

ENV TF_DEV=true
ENV TF_RELEASE=1
ENV GO111MODULE=on

WORKDIR $GOPATH/src/github.com/hashicorp/terraform

RUN go get -u github.com/kardianos/govendor
RUN go get -u golang.org/x/tools/cmd/stringer
RUN go get -u golang.org/x/tools/cmd/cover
# To utilize Docker caching
COPY go.mod go.mod
RUN go mod vendor
COPY . .
RUN /bin/bash scripts/build.sh

WORKDIR $GOPATH
ENTRYPOINT ["terraform"]
