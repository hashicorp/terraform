---
layout: "remotestate"
page_title: "Remote State Backend: local"
sidebar_current: "docs-state-remote-local"
description: |-
  Remote state stored using the local file system.
---

# local

Remote state backend that uses the local file system.

## Example Usage

```
terraform remote config \
    -backend=local \
    -backend-config="path=/path/to/terraform.tfstate"
```

## Example Reference

```
data "terraform_remote_state" "foo" {
    backend = "local"
    config {
        path = "${path.module}/../../terraform.tfstate"
    }
}
```

## Configuration variables

The following configuration options are supported:

 * `path` - (Required) The path to the `tfstate` file.
