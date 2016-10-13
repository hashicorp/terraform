---
layout: "docs"
page_title: "Command: remote config"
sidebar_current: "docs-commands-remote-config"
description: |-
  The `terraform remote config` command is used to configure Terraform to make
  use of remote state storage, change remote storage configuration, or
  to disable it.
---

# Command: remote config

The `terraform remote config` command is used to configure the use of remote state storage. By default, Terraform persists its state to a local disk. When remote state storage is enabled, Terraform will automatically fetch the latest state from the remote server when required. If updates are made, the newest state is persisted back to the remote server. In this mode, users do not need to store the state using version control or shared storage.

## Usage

Usage: `terraform remote config [options]`

The `remote config` command can be used to enable remote storage, change configuration or disable the use of remote storage. Terraform supports multiple types of storage backends, specified by using the `-backend` flag. By default, Atlas is assumed to be the storage backend. Each backend expects different configuration arguments documented below.

When remote storage is enabled, the existing local state file will be migrated. By default, `remote config` will look for the `terraform.tfstate` file, but that can be specified by the `-state` flag. If no state file exists, a blank state will be configured.

When remote storage is disabled, the existing remote state is migrated back to a local file. The location of the new local state file defaults to the path specified in the `-state` flag.

When enabling remote storage, we use the `-backend-config` flag to set any required configuration variables. 

Supported storage backends and supported features of each backend are documented in the [Remote State](/docs/state/remote/index.html) section.

The command-line flags are all optional. The list of available flags are:

* `-backend=Atlas` - The remote backend to use. Must be one of the
  supported backends.

* `-backend-config="k=v"` - Specify a configuration variable for a backend.
  This is how you set any required variables for the backend.

* `-backup=path` - Path to backup the existing state file before
  modifying. Defaults to the "-state" path with ".backup" extension.
  Set to "-" to disable backup.

* `-disable` - Disables remote state management and migrates the state
  to the `-state` path.

* `-pull=true` - Controls if the remote state is pulled before disabling
  or after enabling. This defaults to true to ensure the latest state
  is available under both conditions.

* `-state=path` - Path to read state. Defaults to `terraform.tfstate`
  unless remote state is enabled.

## Example: Consul

This example below will push your remote state to a Consul server. 

```
$ terraform remote config \
    -backend=consul \
    -backend-config="address=consul.example.com:80" \
    -backend-config="path=tf"
```
