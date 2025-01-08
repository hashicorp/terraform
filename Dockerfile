# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: BUSL-1.1

# This Dockerfile builds on golang:alpine by building Terraform from source
# using the current working directory.
#
# This produces a docker image that contains a working Terraform binary along
# with all of its source code. This is not what produces the official releases
# in the "terraform" namespace on Dockerhub; those images include only
# the officially-released binary from releases.hashicorp.com and are
# built by the (closed-source) official release process.

# Pinned tag using SHA
# sha256:c2335038e2230960f81cb2f9f1fc5eca45e23b765de1848c7bbfaebcfd32d90d
# https://github.com/google/go-containerregistry/blob/main/cmid/crane/README.md
FROM docker.mirror.hashicorp.services/golang@sha256:c2335038e2230960f81cb2f9f1fc5eca45e23b765de1848c7bbfaebcfd32d90d
LABEL maintainer="HashiCorp Terraform Team <terraform@hashicorp.com>"

RUN apk add --no-cache git bash openssh

ENV TF_DEV=true
ENV TF_RELEASE=1

WORKDIR $GOPATH/src/github.com/hashicorp/terraform
COPY . .
RUN /bin/bash ./scripts/build.sh

WORKDIR $GOPATH
ENTRYPOINT ["terraform"]
