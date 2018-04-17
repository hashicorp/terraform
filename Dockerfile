# This Dockerfile builds on golang:alpine by building Terraform from source
# using the current working directory.
#
# This produces a docker image that contains a working Terraform binary along
# with all of its source code, which is what gets released on hub.docker.com
# as terraform:full. The main releases (terraform:latest, terraform:light and
# the release tags) are lighter images including only the officially-released
# binary from releases.hashicorp.com; these are built instead from
# scripts/docker-release/Dockerfile-release.

FROM golang:alpine AS builder
LABEL maintainer="HashiCorp Terraform Team <terraform@hashicorp.com>"

RUN apk add --update git bash openssh

ENV TF_DEV=true
ENV TF_RELEASE=1

WORKDIR $GOPATH/src/github.com/hashicorp/terraform
COPY . .
RUN /bin/bash scripts/build.sh

# Take building output and copy it into a clean image

FROM alpine:3.7

RUN apk --no-cache add ca-certificates
RUN apk update && apk upgrade && apk add --no-cache bash

WORKDIR /usr/local/bin

COPY --from=builder "/go/bin/terraform" .

ENTRYPOINT ["terraform"]
