# Building Terraform

This document contains details about the process for building binaries for
Terraform. 

## Versioning

As a pre-1.0 project, we use the MINOR and PATCH versions as follows:

 * a `MINOR` version increment indicates a release that may contain backwards
   incompatible changes
 * a `PATCH` version increment indicates a release that may contain bugfixes as
   well as additive (backwards compatible) features and enhancements

## Process

If only need to build binaries for the platform you're running (Windows, Linux,
Mac OS X etc..), you can follow the instructions in the README for [Developing
Terraform][1].

The guide below outlines the steps HashiCorp takes to build the official release 
binaries for Terraform. This process will generate a set of binaries for each supported
platform, using the [gox](https://github.com/mitchellh/gox) tool.


```sh
# clone the repository if needed
git clone https://github.com/hashicorp/terraform.git
cd terraform

# Verify unit tests pass
make test

# Build the release
# This generates binaries for each platform and places them in the pkg folder
make bin
```

After running these commands, you should have binaries for all supported
platforms in the `pkg` folder.


[1]: https://github.com/hashicorp/terraform#developing-terraform
