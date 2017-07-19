# Terraform Docker Release Build

This directory contains configuration to drive the Dockerhub automated build
for Terraform. This is different than the root Dockerfile (which produces
the "full" image on Dockerhub) because it uses the release archives from
releases.hashicorp.com. It is therefore not possible to use this configuration
to build an image for a commit that hasn't been released.

## How it works

Dockerhub runs the `hooks/build` script to trigger the build. That uses
`git describe` to identify the tag corresponding to the current `HEAD`. If
the current commit _isn't_ tagged with a version number corresponding to
a Terraform release already on releases.hashicorp.com, the build will fail.

## What it produces

This configuration is used to produce the "latest", "light", and "beta"
tags in Dockerhub, as well as specific version tags.

* "latest" and "light" are synonyms, and are built from a branch in this
repository called "stable".
* "beta" is built from a branch called "beta".

All of these branches should be updated only to _tagged_ commits, and only when
it is desirable to create a new release image.

## The `full` and `master` images image

This configuration does not produce the "full" image. That is instead produced
by the `Dockerfile` in the repository root, driven by updates to the "stable"
branch.

The "master" tag is updated for _every_ commit to the master branch of
the Terraform core repository. It is not recommended to use these images for
any production use, but they can be useful for testing bleeding-edge features
that are not yet included in a release.
