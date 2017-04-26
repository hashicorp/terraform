---
layout: "terraform"
page_title: "Terraform: terraform_remote_state"
sidebar_current: "docs-terraform-datasource-remote-state"
description: |-
  Accesses state meta data from a remote backend.
---

# remote_state

Retrieves state meta data from a remote backend

## Example Usage

```hcl
data "terraform_remote_state" "vpc" {
  backend = "atlas"
  config {
    name = "hashicorp/vpc-prod"
  }
}

resource "aws_instance" "foo" {
  # ...
  subnet_id = "${data.terraform_remote_state.vpc.subnet_id}"
}
```

## Argument Reference

The following arguments are supported:

* `backend` - (Required) The remote backend to use.
* `environment` - (Optional) The Terraform environment to use.
* `config` - (Optional) The configuration of the remote backend.
 * Remote state config docs can be found [here](/docs/backends/types/terraform-enterprise.html)

## Attributes Reference

The following attributes are exported:

* `backend` - See Argument Reference above.
* `config` - See Argument Reference above.

In addition, each output in the remote state appears as a top level attribute
on the `terraform_remote_state` resource.

## Root Outputs Only

Only the root level outputs from the remote state are accessible. Outputs from
modules within the state cannot be accessed. If you want a module output to be
accessible via a remote state, you must thread the output through to a root
output.

An example is shown below:

```hcl
module "app" {
  source = "..."
}

output "app_value" {
  value = "${module.app.value}"
}
```

In this example, the output `value` from the "app" module is available as
"app_value". If this root level output hadn't been created, then a remote state
resource wouldn't be able to access the `value` output on the module.
