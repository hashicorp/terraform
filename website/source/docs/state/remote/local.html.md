---
layout: "remotestate"
page_title: "Remote State Backend: local"
sidebar_current: "docs-state-remote-local"
description: |-
  Terraform can store the state locally.
---

# local

This is just a placeholder

## Example Usage

```
terraform remote config \
    -backend=local \
    -backend-config="path=/path/to/terraform.tfstate"
```

## Example Referencing

```
resource "terraform_remote_state" "foo" {
    backend = "local"
    config {
        path = "${path.module}/../../terraform.tfstate"
    }
}
```

## Configuration variables

The following configuration options are supported:

 * `path` - (Required) The path to the `terraform.tfstate` file.
