# This Dockerfile builds on golang:alpine by building Terraform from source
# using the current working directory.
#
# This produces a docker image that contains a working Terraform binary along
# with all of its source code, which is what gets released on hub.docker.com
# as terraform:full. The main releases (terraform:latest, terraform:light and
# the release tags) are lighter images including only the officially-released
# binary from releases.hashicorp.com; these are built instead from
# scripts/docker-release/Dockerfile-release.

FROM golang:alpine AS build
LABEL maintainer="HashiCorp Terraform Team <terraform@hashicorp.com>"

ENV TF_DEV=true
ENV TF_RELEASE=1

RUN apk add --no-cache git bash openssh

WORKDIR $GOPATH/src/github.com/hashicorp/terraform
COPY . .
RUN /bin/bash scripts/build.sh

FROM golang:alpine AS final

RUN apk add --no-cache git openssh

COPY --from=build ["${GOPATH}/bin/terraform", "/bin/terraform"]

ENTRYPOINT ["/bin/terraform"]
