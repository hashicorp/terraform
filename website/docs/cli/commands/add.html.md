---
layout: "docs"
page_title: "Command: add"
sidebar_current: "docs-commands-add"
description: |-
  The `terraform add` command generates resource configuration templates.
---

# Command: add

The `terraform add` command generates a resource configuration template with
`null` placeholder values for all attributes, unless the `-from-state` flag is
used. By default, the template only includes required resource attributes; the
`-optional` flag tells Terraform to also include any optional attributes. 

When `terraform add` used with the `-from-state` will _not_ print sensitive
values. You can use `terraform show ADDRESS` to see all values, including
sensitive values, recorded in state for the given resource address.

## Usage

Usage: `terraform add [options] ADDRESS`

This command requires an address that points to a resource which does not
already exist in the configuration. Addresses are in 
[resource addressing format](/docs/cli/state/resource-addressing.html).

This command accepts the following options:

`-from-state` - populate the template with values from a resource
already in state. Sensitive values are redacted.

`-optional` - include optional attributes in the template.

`-out=FILENAME` - writes the template to the given filename. If the file already
exists, the template will be added to the end of the file.

`-provider=provider` - override the configured provider for the resource. By
default, Terraform will use the configured provider for the given resource type,
and that is the best behavior in most cases.
