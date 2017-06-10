---
layout: "intro"
page_title: "Installing Terraform"
sidebar_current: "gettingstarted-install"
description: |-
  Terraform must first be installed on your machine. Terraform is distributed as
  a binary package for all supported platforms and architecture. This page will
  not cover how to compile Terraform from source.
---

# Install Terraform

Terraform must first be installed on your machine. Terraform is distributed as a
[binary package](/downloads.html) for all supported platforms and architectures.
This page will not cover how to compile Terraform from source, but compiling
from source is covered in the [documentation](/docs/index.html) for those who
want to be sure they're compiling source they trust into the final binary.

## Installing Terraform

To install Terraform, find the [appropriate package](/downloads.html) for your
system and download it. Terraform is packaged as a zip archive.

After downloading Terraform, unzip the package. Terraform runs as a single
binary named `terraform`. Any other files in the package can be safely removed
and Terraform will still function.

The final step is to make sure that the `terraform` binary is available on the `PATH`.
See [this page](https://stackoverflow.com/questions/14637979/how-to-permanently-set-path-on-linux)
for instructions on setting the PATH on Linux and Mac.
[This page](https://stackoverflow.com/questions/1618280/where-can-i-set-path-to-make-exe-on-windows)
contains instructions for setting the PATH on Windows.

## Verifying the Installation

After installing Terraform, verify the installation worked by opening a new
terminal session and checking that `terraform` is available. By executing
`terraform` you should see help output similar to this:

```text
$ terraform
Usage: terraform [--version] [--help] <command> [args]

The available commands for execution are listed below.
The most common, useful commands are shown first, followed by
less common or more advanced commands. If you're just getting
started with Terraform, stick with the common commands. For the
other commands, please read the help and docs before usage.

Common commands:
    apply              Builds or changes infrastructure
    console            Interactive console for Terraform interpolations
# ...
```

If you get an error that `terraform` could not be found, your `PATH` environment
variable was not set up properly. Please go back and ensure that your `PATH`
variable contains the directory where Terraform was installed.

## Next Steps

Time to [build infrastructure](/intro/getting-started/build.html) using a
minimal Terraform configuration file. You will be able to examine Terraform's
execution plan before you deploy it to AWS.
