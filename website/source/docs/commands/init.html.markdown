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

* `-address=url` - URL of the remote storage server. Required for HTTP backend,
  optional for Atlas and Consul.

* `-access-token=token` - Authentication token for state storage server.
  Required for Atlas backend, optional for Consul.

* `-backend=atlas` - Specifies the type of remote backend. Must be one
  of Atlas, Consul, or HTTP. Defaults to atlas.

* `-name=name` - Name of the state file in the state storage server.
  Required for Atlas backend.

* `-path=path` - Path of the remote state in Consul. Required for the Consul backend.

