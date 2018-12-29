---
layout: "docs"
page_title: "Command: taint"
sidebar_current: "docs-commands-taint"
description: |-
  The `terraform taint` command manually marks a Terraform-managed resource as tainted, forcing it to be destroyed and recreated on the next apply.
---

# Command: taint

The `terraform taint` command manually marks a Terraform-managed resource
as tainted, forcing it to be destroyed and recreated on the next apply.

This command _will not_ modify infrastructure, but does modify the
state file in order to mark a resource as tainted. Once a resource is
marked as tainted, the next
[plan](/docs/commands/plan.html) will show that the resource will
be destroyed and recreated and the next
[apply](/docs/commands/apply.html) will implement this change.

Forcing the recreation of a resource is useful when you want a certain
side effect of recreation that is not visible in the attributes of a resource.
For example: re-running provisioners will cause the node to be different
or rebooting the machine from a base image will cause new startup scripts
to run.

Note that tainting a resource for recreation may affect resources that
depend on the newly tainted resource. For example, a DNS resource that
uses the IP address of a server may need to be modified to reflect
the potentially new IP address of a tainted server. The
[plan command](/docs/commands/plan.html) will show this if this is
the case.

## Usage

Usage: `terraform taint [options] name`

The `name` argument is the name of the resource to mark as tainted.
The format of this argument is `TYPE.NAME`, such as `aws_instance.foo`.

The command-line flags are all optional. The list of available flags are:

* `-allow-missing` - If specified, the command will succeed (exit code 0)
    even if the resource is missing. The command can still error, but only
    in critically erroneous cases.

* `-backup=path` - Path to the backup file. Defaults to `-state-out` with
  the ".backup" extension. Disabled by setting to "-".

* `-lock=true` - Lock the state file when locking is supported.

* `-lock-timeout=0s` - Duration to retry a state lock.

* `-module=path` - The module path where the resource to taint exists.
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
