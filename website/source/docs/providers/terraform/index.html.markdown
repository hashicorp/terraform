---
layout: "terraform"
page_title: "Provider: Terraform"
sidebar_current: "docs-terraform-index"
description: |-
  The Terraform provider is used to access meta data from shared infrastructure.
---

# Terraform Provider

The terraform provider exposes resources to access state meta data
for Terraform outputs from shared infrastructure.

The terraform provider is what we call a _logical provider_. This has no
impact on how it behaves, but conceptually it is important to understand.
The terraform provider doesn't manage any _physical_ resources; it isn't
creating servers, writing files, etc. It is used to access the outputs
of other Terraform states to be used as inputs for resources.
Examples will explain this best.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Shared infrastructure state stored in Atlas
resource "terraform_remote_state" "vpc" {
    backend = "atlas"
    config {
        path = "hashicorp/vpc-prod"
    }
}

resource "aws_instance" "foo" {
    # ...
    subnet_id = "${terraform_remote_state.vpc.output.subnet_id}"
}
```
