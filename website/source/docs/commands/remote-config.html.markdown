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

When enabling remote storage, use the `-backend-config` flag to set
the required configuration variables as documented below. See the example
below this section for more details.

When remote storage is disabled, the existing remote state is migrated
to a local file. This defaults to the `-state` path during restore.

The following backends are supported:

* Atlas - Stores the state in Atlas. Requires the `name` and `access_token`
  variables. The `address` variable can optionally be provided.

* Consul - Stores the state in the KV store at a given path.
  Requires the `path` variable. The `address` and `access_token`
  variables can optionally be provided. Address is assumed to be the
  local agent if not provided.

* S3 - Stores the state as a given key in a given bucket on Amazon S3.
  Requires the `bucket` and `key` variables. Supports and honors the standard
  AWS environment variables `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`
  and `AWS_DEFAULT_REGION`. These can optionally be provided as parameters
  in the `access_key`, `secret_key` and `region` variables
  respectively, but passing credentials this way is not recommended since they
  will be included in cleartext inside the persisted state.

* HTTP - Stores the state using a simple REST client. State will be fetched
  via GET, updated via POST, and purged with DELETE. Requires the `address` variable.

The command-line flags are all optional. The list of available flags are:

* `-backend=Atlas` - The remote backend to use. Must be one of the above
  supported backends.

* `-backend-config="k=v"` - Specify a configuration variable for a backend.
  This is how you set the required variables for the backends above.

* `-backup=path` - Path to backup the existing state file before
  modifying. Defaults to the "-state" path with ".backup" extension.
  Set to "-" to disable backup.

* `-disable` - Disables remote state management and migrates the state
  to the `-state` path.

* `-pull=true` - Controls if the remote state is pulled before disabling
  or after enabling. This defaults to true to ensure the latest state
  is available under both conditions.

* `-state=path` - Path to read state. Defaults to "terraform.tfstate"
  unless remote state is enabled.

## Example: Consul

The example below will push your remote state to Consul. Note that for
this example, it would go to the public Consul demo. In practice, you
should use your own private Consul server:

```
$ terraform remote config \
    -backend=consul \
    -backend-config="address=demo.consul.io:80" \
    -backend-config="path=tf" \
    -backend-config="access_token=1234567890abcdefgh1234567890"
```
