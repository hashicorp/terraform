---
layout: "intro"
page_title: "Installing Terraform"
sidebar_current: "gettingstarted-install"
description: |-
  Terraform must first be installed on your machine. Terraform is distributed as a binary package for all supported platforms and architecture. This page will not cover how to compile Terraform from source.
---

# Install Terraform

Terraform must first be installed on your machine. Terraform is distributed
as a [binary package](/downloads.html) for all supported platforms and
architecture. This page will not cover how to compile Terraform from
source.

## Installing Terraform

To install Terraform, find the [appropriate package](/downloads.html) for
your system and download it. Terraform is packaged as a zip archive.

After downloading Terraform, unzip the package into a directory where
Terraform will be installed. The directory will contain a set of binary
programs, such as `terraform`, `terraform-provider-aws`, etc. The final
step is to make sure the directory you installed Terraform to is on the
PATH. See
[this page](https://stackoverflow.com/questions/14637979/how-to-permanently-set-path-on-linux)
for instructions on setting the PATH on Linux and Mac.
[This page](https://stackoverflow.com/questions/1618280/where-can-i-set-path-to-make-exe-on-windows)
contains instructions for setting the PATH on Windows.

Example for Linux/Mac - Type the following into your terminal:
>`PATH=/usr/local/terraform/bin:/home/your-user-name/terraform:$PATH`

Example for Windows - Type the following into Powershell:
>`set PATH=%PATH%;C:\terraform`


## Verifying the Installation

After installing Terraform, verify the installation worked by opening a new
terminal session and checking that `terraform` is available. By executing
`terraform` you should see help output similar to that below:

```
$ terraform
usage: terraform [--version] [--help] <command> [<args>]

Available commands are:
    apply       Builds or changes infrastructure
    destroy     Destroy Terraform-managed infrastructure
    get         Download and install modules for the configuration
    graph       Create a visual graph of Terraform resources
    init        Initializes Terraform configuration from a module
    output      Read an output from a state file
    plan        Generate and show an execution plan
    push        Upload this Terraform module to Atlas to run
    refresh     Update local state file against real resources
    remote      Configure remote state storage
    show        Inspect Terraform state or plan
    taint       Manually mark a resource for recreation
    validate    Validates the Terraform files
    version     Prints the Terraform version
```

If you get an error that `terraform` could not be found, then your PATH
environment variable was not setup properly. Please go back and ensure
that your PATH variable contains the directory where Terraform was installed.

Otherwise, Terraform is installed and ready to go! Nice!

## Next Step

Time to [build infrastructure](/intro/getting-started/build.html)
using a minimal Terraform configuration file. You will be able to
examine Terraform's execution plan before you deploy it to AWS.
