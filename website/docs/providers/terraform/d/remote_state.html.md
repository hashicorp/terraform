---
layout: "terraform"
page_title: "Terraform: terraform_remote_state"
sidebar_current: "docs-terraform-datasource-remote-state"
description: |-
  Accesses state meta data from a remote backend.
---

# remote_state

[backends]: /docs/backends/index.html

Retrieves state data from a [Terraform backend][backends]. This allows you to
use the root-level outputs of one or more Terraform configurations as input data
for another configuration.

Although this data source uses Terraform's [backends][], it doesn't have the
same limitations as the main backend configuration. You can use any number of
`remote_state` data sources with differently configured backends, and you can
use interpolations when configuring them.

## Example Usage

```hcl
data "terraform_remote_state" "vpc" {
  backend = "remote"

  config = {
    organization = "hashicorp"
    workspaces = {
      name = "vpc-prod"
    }
  }
}

# Terraform >= 0.12
resource "aws_instance" "foo" {
  # ...
  subnet_id = data.terraform_remote_state.vpc.outputs.subnet_id
}

# Terraform <= 0.11
resource "aws_instance" "foo" {
  # ...
  subnet_id = "${data.terraform_remote_state.vpc.subnet_id}"
}
```

## Argument Reference

The following arguments are supported:

* `backend` - (Required) The remote backend to use.
* `workspace` - (Optional) The Terraform workspace to use, if the backend
  supports workspaces.
* `config` - (Optional; object) The configuration of the remote backend.
  Although this argument is listed as optional, most backends require
  some configuration.

    The `config` object can use any arguments that would be valid in the
    equivalent `terraform { backend "<TYPE>" { ... } }` block. See
    [the documentation of your chosen backend](/docs/backends/types/index.html)
    for details.

    -> **Note:** If the backend configuration requires a nested block, specify
    it here as a normal attribute with an object value. (For example,
    `workspaces = { ... }` instead of `workspaces { ... }`.)
* `defaults` - (Optional; object) Default values for outputs, in case the state
  file is empty or lacks a required output.

## Attributes Reference

In addition to the above, the following attributes are exported:

* (v0.12+) `outputs` - An object containing every root-level
  [output](/docs/configuration/outputs.html) in the remote state.
* (<= v0.11) `<OUTPUT NAME>` - Each root-level [output](/docs/configuration/outputs.html)
  in the remote state appears as a top level attribute on the data source.

## Root Outputs Only

Only the root-level outputs from the remote state are accessible. Outputs from
modules within the state cannot be accessed. If you want a module output or a
resource attribute to be accessible via a remote state, you must thread the
output through to a root output.

For example:

```hcl
module "app" {
  source = "..."
}

output "app_value" {
  value = "${module.app.value}"
}
```

In this example, the output `value` from the "app" module is available as
`app_value`. If this root level output hadn't been created, then a remote state
resource wouldn't be able to access the `value` output on the module.
