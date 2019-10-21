# This Dockerfile is not intended for general use, but is rather used to
# package up official Terraform releases (from releases.hashicorp.com) to
# release on Dockerhub as the "light" release images.
#
# The main Dockerfile in the root of the repository is more generally-useful,
# since it is able to build a docker image of the current state of the work
# tree, without any dependency on there being an existing release on
# releases.hashicorp.com.

FROM alpine:3.9.2 as build
LABEL maintainer="HashiCorp Terraform Team <terraform@hashicorp.com>"

# This is intended to be run from the hooks/build script, which sets this
# appropriately based on git tags.
ARG TERRAFORM_VERSION=UNSPECIFIED

COPY releases_public_key .

# What's going on here?
# - Download the indicated release along with its checksums and signature for the checksums
# - Verify that the checksums file is signed by the Hashicorp releases key
# - Verify that the zip file matches the expected checksum
# - Extract the zip file so it can be run

RUN apk add --no-cache git curl openssh gnupg && \
    curl -O https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip && \
    curl -O https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_SHA256SUMS.sig && \
    curl -O https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_SHA256SUMS && \
    gpg --import releases_public_key && \
    gpg --verify terraform_${TERRAFORM_VERSION}_SHA256SUMS.sig terraform_${TERRAFORM_VERSION}_SHA256SUMS && \
    grep linux_amd64 terraform_${TERRAFORM_VERSION}_SHA256SUMS >terraform_${TERRAFORM_VERSION}_SHA256SUMS_linux_amd64 && \
    sha256sum -cs terraform_${TERRAFORM_VERSION}_SHA256SUMS_linux_amd64 && \
    unzip terraform_${TERRAFORM_VERSION}_linux_amd64.zip -d /bin && \
    rm -f terraform_${TERRAFORM_VERSION}_linux_amd64.zip terraform_${TERRAFORM_VERSION}_SHA256SUMS*

FROM alpine:3.9.2 as final

LABEL "com.hashicorp.terraform.version"="${TERRAFORM_VERSION}"

RUN apk add --no-cache git openssh

COPY --from=build ["/bin/terraform", "/bin/terraform"]

ENTRYPOINT ["/bin/terraform"]
