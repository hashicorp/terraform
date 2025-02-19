# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: BUSL-1.1

# This Dockerfile is not intended for general use, but is rather used to
# produce our "light" release packages as part of our official release
# pipeline.
#
# If you want to test this locally you'll need to set the three arguments
# to values realistic for what the hashicorp/actions-docker-build GitHub
# action would set, and ensure that there's a suitable "terraform" executable
# in the dist/linux/${TARGETARCH} directory.

FROM docker.mirror.hashicorp.services/alpine:latest AS default

# This is intended to be run from the hashicorp/actions-docker-build GitHub
# action, which sets these appropriately based on context.
ARG PRODUCT_VERSION=UNSPECIFIED
ARG PRODUCT_REVISION=UNSPECIFIED
ARG BIN_NAME=terraform

# This argument is set by the Docker toolchain itself, to the name
# of the CPU architecture we're building an image for.
# Our caller should've extracted the corresponding "terraform" executable
# into dist/linux/${TARGETARCH} for us to use.
ARG TARGETARCH

LABEL maintainer="HashiCorp Terraform Team <terraform@hashicorp.com>"

# New standard version label.
LABEL version=$PRODUCT_VERSION

# Historical Terraform-specific label preserved for backward compatibility.
LABEL "com.hashicorp.terraform.version"="${PRODUCT_VERSION}"

# @see https://specs.opencontainers.org/image-spec/annotations/?v=v1.0.1#pre-defined-annotation-keys
LABEL org.opencontainers.image.title=${BIN_NAME} \
      org.opencontainers.image.description="Terraform enables you to safely and predictably create, change, and improve infrastructure" \
      org.opencontainers.image.authors="HashiCorp Terraform Team <terraform@hashicorp.com>" \
      org.opencontainers.image.url="https://www.terraform.io/" \
      org.opencontainers.image.documentation="https://www.terraform.io/docs" \
      org.opencontainers.image.source="https://github.com/hashicorp/terraform" \
      org.opencontainers.image.version=${PRODUCT_VERSION} \
      org.opencontainers.image.revision=${PRODUCT_REVISION} \
      org.opencontainers.image.vendor="HashiCorp" \
      org.opencontainers.image.licenses="BUSL-1.1"

RUN apk add --no-cache git openssh ca-certificates

# Copy the license file as per Legal requirement
COPY LICENSE "/usr/share/doc/${BIN_NAME}/LICENSE.txt"

# The hashicorp/actions-docker-build GitHub Action extracts the appropriate
# release package for our target architecture into the current working
# directory before running "docker build", which we'll then copy into the
# Docker image to make sure that we use an identical binary as all of the
# other official release channels.
COPY ["dist/linux/${TARGETARCH}/terraform", "/bin/terraform"]

ENTRYPOINT ["/bin/terraform"]
