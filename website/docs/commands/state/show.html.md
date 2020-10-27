---
layout: "docs"
page_title: "Command: state show"
sidebar_current: "docs-commands-state-sub-show"
description: |-
  The `terraform state show` command is used to show the attributes of a single resource in the Terraform state.
---

# Command: state show

The `terraform state show` command is used to show the attributes of a
single resource in the
[Terraform state](/docs/state/index.html).

## Usage

Usage: `terraform state show [options] ADDRESS`

The command will show the attributes of a single resource in the
state file that matches the given address.

This command requires an address that points to a single resource in the
state. Addresses are
in [resource addressing format](/docs/commands/state/addressing.html).

The command-line flags are all optional. The list of available flags are:

* `-state=path` - Path to the state file. Defaults to "terraform.tfstate".
  Ignored when [remote state](/docs/state/remote.html) is used.

The output of `terraform state show` is intended for human consumption, not
programmatic consumption. To extract state data for use in other software, use
[`terraform show -json`](../show.html#json-output) and decode the result
using the documented structure.

## Example: Show a Resource

The example below shows a `packet_device` resource named `worker`:

```
$ terraform state show 'packet_device.worker'
# packet_device.worker:
resource "packet_device" "worker" {
    billing_cycle = "hourly"
    created       = "2015-12-17T00:06:56Z"
    facility      = "ewr1"
    hostname      = "prod-xyz01"
    id            = "6015bg2b-b8c4-4925-aad2-f0671d5d3b13"
    locked        = false
}
```

## Example: Show a Module Resource

The example below shows a `packet_device` resource named `worker` inside a module named `foo`:

```shell
$ terraform state show 'module.foo.packet_device.worker'
```

## Example: Show a Resource configured with count

The example below shows the first instance of a `packet_device` resource named `worker` configured with
[`count`](/docs/configuration/resources.html#count-multiple-resource-instances-by-count):

```shell
$ terraform state show 'packet_device.worker[0]'
```

## Example: Show a Resource configured with for_each

The example below shows the `"example"` instance of a `packet_device` resource named `worker` configured with
[`for_each`](/docs/configuration/resources.html#for_each-multiple-resource-instances-defined-by-a-map-or-set-of-strings):

Linux, Mac OS, and UNIX:

```shell
$ terraform state show 'packet_device.worker["example"]'
```

PowerShell:

```shell
$ terraform state show 'packet_device.worker[\"example\"]'
```

Windows `cmd.exe`:

```shell
$ terraform state show packet_device.worker[\"example\"]
```
