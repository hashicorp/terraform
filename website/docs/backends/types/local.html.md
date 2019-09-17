---
layout: "backend-types"
page_title: "Backend Type: local"
sidebar_current: "docs-backends-types-enhanced-local"
description: |-
  Terraform can store the state remotely, making it easier to version and work with in a team.
---

# local

**Kind: Enhanced**

The local backend stores state on the local filesystem, locks that
state using system APIs, and performs operations locally.

## Example Configuration

```hcl
terraform {
  backend "local" {
    path = "relative/path/to/terraform.tfstate"
  }
}
```

## Example Reference

```hcl
data "terraform_remote_state" "foo" {
  backend = "local"

  config = {
    path = "${path.module}/../../terraform.tfstate"
  }
}
```

## Configuration variables

The following configuration options are supported:

 * `path` - (Optional) The path to the `tfstate` file. This defaults to
   "terraform.tfstate" relative to the root module by default.
 * `workspace_dir` - (Optional) The path to non-default workspaces.
