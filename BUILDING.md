# Building Terraform

This document contains details about the process for building release-style
binaries for Terraform.

(If you are intending instead to make changes to Terraform and build binaries
only for your local testing, see
[the contributing guide](.github/CONTRIBUTING.md).)

## Versioning

Until Terraform v1.0, Terraform's versioning scheme is as follows:

* Full version strings start with a zero in the initial position.
* The second position increments for _major_ releases, which may contain
  backwards incompatible changes.
* The third and final position increments for _minor_ releases, which
  we aim to keep backwards compatible with prior releases for the same major
  version.

Although the Terraform team takes care to preserve compatibility between
major releases as much as possible, major release upgrades will often require
specific upgrade actions for a subset of users as we refine the product
design in preparation for making more specific backward-compatibility promises
in a later Terraform 1.0 release.

## Process

Terraform release binaries are built via cross-compilation on a Linux
system, using [gox](https://github.com/mitchellh/gox). 

The steps below are a subset of the steps HashiCorp uses to prepare the
official distribution packages available from
[the download page](https://www.terraform.io/downloads.html). This
process will generate an executable for each of the supported target platforms.

HashiCorp prepares release binaries on Linux amd64 systems. This build process
may need to be adjusted for other host platforms.

```sh
# clone the repository if needed
git clone https://github.com/hashicorp/terraform.git
cd terraform

# Verify that the unit tests are passing
make test

# Run preparation steps and then build the executable for each target platform
# in the subdirectory "pkg".
# This generates binaries for each platform and places them in the pkg folder
make bin
```

Official releases are subsequently then packaged, hashed, and signed before
uploading to [the HashiCorp releases service](https://releases.hashicorp.com/terraform/).
Those final packaging steps are not fully reproducible using the contents
of this repository due to the use of HashiCorp's private signing key. However,
you can place the generated executables in `.zip` archives to produce a
similar result without the checksums and digital signature.

## Release Bundles for use in Terraform Enterprise

If you wish to build distribution archives that blend official Terraform
release executables with a mixture of official and third-party provider builds,
see [the `terraform-bundle` tool](tools/terraform-bundle).
