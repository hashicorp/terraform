---
layout: "docs"
page_title: "Command: init"
sidebar_current: "docs-commands-init"
description: |-
  The `terraform init` command is used to initialize a Terraform configuration using another module as a skeleton.
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

The command-line options available are a subset of the ones for the
[remote command](/docs/commands/remote.html), and are used to initialize
a remote state configuration if provided.

The command-line flags are all optional. The list of available flags are:

* `-backend=atlas` - Specifies the type of remote backend. Must be one
  of Atlas, Consul, S3, or HTTP. Defaults to Atlas.

* `-backend-config="k=v"` - Specify a configuration variable for a backend. This is how you set the required variables for the selected backend (as detailed in the [remote command documentation](/docs/commands/remote.html).


## Example: Consul

This example will initialize the current directory and configure Consul remote storage:

```
$ terraform init \
    -backend=consul \
    -backend-config="address=your.consul.endpoint:443" \
    -backend-config="scheme=https" \
    -backend-config="path=tf/path/for/project" \
    /path/to/source/module
```

## Example: S3

This example will initialize the current directory and configure S3 remote storage:

```
$ terraform init \
    -backend=s3 \
    -backend-config="bucket=your-s3-bucket" \
    -backend-config="key=tf/path/for/project.json" \
    -backend-config="acl=bucket-owner-full-control" \
    /path/to/source/module
```
