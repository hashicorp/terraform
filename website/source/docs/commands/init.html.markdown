---
layout: "docs"
page_title: "Command: init"
sidebar_current: "docs-commands-init"
---

# Command: init

The `terraform init` command is used to initialize a Terraform configuration
using another
[module](/docs/modules/index.html)
as a skeleton.

## Usage

Usage: `terraform init [options] SOURCE [DIR]`

Init will download the module from SOURCE and copy it into the DIR
(which defaults to the current working directory). Version control
information from the module (such as Git history) will not be copied.

The directory being initialized must be empty of all Terraform configurations.
If the module has other files which conflict with what is already in the
directory, they _will be overwritten_.
