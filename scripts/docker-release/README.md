# Terraform Docker Release Build

This directory contains configuration to drive the docker image releases for
Terraform.

Two different types of image are produced for each Terraform release:

* A "light" image that includes just the release binary that should match
  what's on releases.hashicorp.com.

* A "full" image that contains all of the Terraform source code and a binary
  built from that source.

The latter can be produced for any arbitrary commit by running `docker build`
in the root of this repository. The former requires that the release archive
already be deployed on releases.hashicorp.com.

## Build and Release

The scripts in this directory are intended for running the steps to build,
tag, and push the two images for a tagged and released version of Terraform.
They expect to be run with git `HEAD` pointed at a release tag, whose name
is used to determine the version to build. The version number indicated
by the tag that `HEAD` is pointed at will be referred to below as
the _current version_.

* `build.sh` builds locally both of the images for the current version.
  This operates on the local docker daemon only, and produces tags that
  include the current version number.

* `tag.sh` updates the `latest`, `light` and `full` tags to refer to the
  images for the current version, which must've been already produced by
  an earlier run of `build.sh`. This operates on the local docker daemon
  only.

* `push.sh` pushes the current version tag and the `latest`, `light` and
  `full` tags up to dockerhub for public consumption. This writes images
  to dockerhub, and so it requires docker credentials that have access to
  write into the `hashicorp/terraform` repository.

### Releasing a new "latest" version

In the common case where a release is going to be considered the new latest
stable version of Terraform, the helper script `release.sh` orchestrates
all of the necessary steps to release to dockerhub:

```
$ git checkout v0.10.0
$ scripts/docker-release/release.sh
```

Behind the scenes this script is running `build.sh`, `tag.sh` and `push.sh`
as described above, with some extra confirmation steps to verify the
correctness of the build.

This script is interactive and so isn't suitable for running in automation.
For automation, run the individual scripts directly.

### Releasing a beta version or a patch to an earlier minor release

The `release.sh` wrapper is not appropriate in two less common situations:

* The version being released is a beta or other pre-release version, with
  a version number like `v0.10.0-beta1` or `v0.10.0-rc1`.

* The version being released belongs to a non-current minor release. For
  example, if the current stable version is `v0.10.1` but the version
  being released is `v0.9.14`.

In both of these cases, only the specific version tag should be updated,
which can be done as follows:

```
$ git checkout v0.11.0-beta1
$ scripts/docker-release/build.sh
$ docker push hashicorp/terraform:0.11.0-beta1
```
