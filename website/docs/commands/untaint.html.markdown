---
layout: "docs"
page_title: "Command: untaint"
sidebar_current: "docs-commands-untaint"
description: |-
  The `terraform untaint` command manually unmarks a Terraform-managed resource as tainted, restoring it as the primary instance in the state.
---

# Command: untaint

The `terraform untaint` command manually unmarks a Terraform-managed resource
as tainted, restoring it as the primary instance in the state. This reverses
either a manual `terraform taint` or the result of provisioners failing on a
resource.

This command _will not_ modify infrastructure, but does modify the state file
in order to unmark a resource as tainted.

~> **NOTE on Tainted Indexes:** In certain edge cases, more than one tainted
instance can be present for a single resource. When this happens, the `-index`
flag is required to select which of the tainted instances to restore as
primary. You can use the `terraform show` command to inspect the state and
determine which index holds the instance you'd like to restore. In the vast
majority of cases, there will only be one tainted instance, and the `-index`
flag can be omitted.

## Usage

Usage: `terraform untaint [options] name`

The `name` argument is the name of the resource to mark as untainted.  The
format of this argument is `TYPE.NAME`, such as `aws_instance.foo`.

The command-line flags are all optional (with the exception of `-index` in
certain cases, see above note). The list of available flags are:

* `-allow-missing` - If specified, the command will succeed (exit code 0)
    even if the resource is missing. The command can still error, but only
    in critically erroneous cases.

* `-backup=path` - Path to the backup file. Defaults to `-state-out` with
  the ".backup" extension. Disabled by setting to "-".

* `-index=n` - Selects a single tainted instance when there are more than one
  tainted instances present in the state for a given resource. This flag is
  required when multiple tainted instances are present. The vast majority of the
  time, there is a maximum of one tainted instance per resource, so this flag
  can be safely omitted.

* `-lock=true` - Lock the state file when locking is supported.

* `-lock-timeout=0s` - Duration to retry a state lock.

* `-module=path` - The module path where the resource to untaint exists.
    By default this is the root path. Other modules can be specified by
    a period-separated list. Example: "foo" would reference the module
    "foo" but "foo.bar" would reference the "bar" module in the "foo"
    module.

* `-no-color` - Disables output with coloring

* `-state=path` - Path to read and write the state file to. Defaults to "terraform.tfstate".
  Ignored when [remote state](/docs/state/remote.html) is used.

* `-state-out=path` - Path to write updated state file. By default, the
  `-state` path will be used. Ignored when
  [remote state](/docs/state/remote.html) is used.
