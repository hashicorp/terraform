---
layout: "terraform"
page_title: "Provider: Terraform"
sidebar_current: "docs-terraform-index"
description: |-
  The Terraform provider is used to access meta data from shared infrastructure.
---

# Terraform Provider

The terraform provider provides access to outputs from the Terraform state
of shared infrastructure.

Use the navigation to the left to read about the available data sources.

## Example Usage

```hcl
# Shared infrastructure state stored in Atlas
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
