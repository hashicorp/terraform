---
layout: "intro"
page_title: "Installing Terraform"
sidebar_current: "gettingstarted-install"
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
[this page](http://stackoverflow.com/questions/14637979/how-to-permanently-set-path-on-linux)
for instructions on setting the PATH on Linux and Mac.
[This page](http://stackoverflow.com/questions/1618280/where-can-i-set-path-to-make-exe-on-windows)
contains instructions for setting the PATH on Windows.

## Verifying the Installation

After installing Terraform, verify the installation worked by opening a new
terminal session and checking that `terraform` is available. By executing
`terraform` you should see help output similar to that below:

```
$ terraform
usage: terraform [--version] [--help] <command> [<args>]

Available commands are:
    apply      Builds or changes infrastructure
    graph      Create a visual graph of Terraform resources
    output     Read an output from a state file
    plan       Generate and show an execution plan
    refresh    Update local state file against real resources
    show       Inspect Terraform state or plan
    version    Prints the Terraform version
```

If you get an error that `terraform` could not be found, then your PATH
environment variable was not setup properly. Please go back and ensure
that your PATH variable contains the directory where Terraform was installed.

Otherwise, Terraform is installed and ready to go!
