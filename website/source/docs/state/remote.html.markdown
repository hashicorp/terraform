---
layout: "docs"
page_title: "Terraform Remote State"
sidebar_current: "docs-modules-remote"
description: |-
  Remote states are a way to access only the outputs of an independent Terraform run.
---

# Terraform Remote State

Remote states are a way to access only the outputs
of an independent Terraform run.

Remote states are a way for teams within an organization to
share infrastructure resources in a read-only way without 
other teams being able to build or modify that infrastructure.
An example: a team builds and maintains a highly-available 
database cluster, and other teams can access the URL and access 
information via remote state without ever risking modifying that infrastructure.

Remote states are accessed as a standard Terraform resource. An example is shown below:

## An Example

The `terraform_remote_state` resource first pulls information from an indepenedent
Terraform state, which can then be used to configure
resources in the current Terraform configuration.

```
resource "terraform_remote_state" "vpc" {
    backend = "atlas"
    config {
        path = "hashicorp/vpc-prod"
    }
}

resource "aws_instance" "foo" {
    # ...
    subnet_id = "${terraform_state.vpc.output.subnet_id}"
}
```
As you can see from the above example, this allows separate teams
to maintain different parts of the infrastructure, and
for other teams to access these. 

## Argument Reference

The following arguments are supported:

* `backend` - (Required) The backend storing the Terraform state. Allowed
values are `atlas`, `http`, `consul`.

* `config` - (Required) Configuration options for the respective backend.
