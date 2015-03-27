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

The `terraform remote config` command is used to configure use of remote
state storage. By default, Terraform persists its state only to a local
disk. When remote state storage is enabled, Terraform will automatically
fetch the latest state from the remote server when necessary and if any
updates are made, the newest state is persisted back to the remote server.
In this mode, users do not need to durably store the state using version
control or shared storaged.

## Usage

Usage: `terraform remote config [options]`

The `remote config` command can be used to enable remote storage, change
configuration or disable the use of remote storage. Terraform supports multiple types
of storage backends, specified by using the `-backend` flag. By default,
Atlas is assumed to be the storage backend. Each backend expects different,
configuration arguments documented below.

When remote storage is enabled, an existing local state file can be migrated.
By default, `remote config` will look for the "terraform.tfstate" file, but that
can be specified by the `-state` flag. If no state file exists, a blank
state will be configured.

When remote storage is disabled, the existing remote state is migrated
to a local file. This defaults to the `-state` path during restore.

The following backends are supported:

* Atlas - Stores the state in Atlas. Requires the `-name` and `-access-token` flag.
  The `-address` flag can optionally be provided.

* Consul - Stores the state in the KV store at a given path.
  Requires the `path` flag. The `-address` and `-access-token`
  flag can optionally be provided. Address is assumed to be the
  local agent if not provided.

* HTTP - Stores the state using a simple REST client. State will be fetched
  via GET, updated via POST, and purged with DELETE. Requires the `-address` flag.

The command-line flags are all optional. The list of available flags are:

* `-address=url` - URL of the remote storage server. Required for HTTP backend,
  optional for Atlas and Consul.

* `-access-token=token` - Authentication token for state storage server.
  Required for Atlas backend, optional for Consul.

* `-backend=Atlas` - Specifies the type of remote backend. Must be one
  of Atlas, Consul, or HTTP. Defaults to Atlas.

* `-backup=path` - Path to backup the existing state file before
  modifying. Defaults to the "-state" path with ".backup" extension.
  Set to "-" to disable backup.

* `-disable` - Disables remote state management and migrates the state
  to the `-state` path.

* `-name=name` - Name of the state file in the state storage server.
  Required for Atlas backend.

* `-path=path` - Path of the remote state in Consul. Required for the
  Consul backend.

* `-pull=true` - Controls if the remote state is pulled before disabling
  or after enabling. This defaults to true to ensure the latest state
  is available under both conditions.

* `-state=path` - Path to read state. Defaults to "terraform.tfstate"
  unless remote state is enabled.

