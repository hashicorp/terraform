---
layout: "dockercloud"
page_title: "Provider: Docker Cloud"
sidebar_current: "docs-dockercloud-index"
description: |-
  The Docker Cloud provider is used to interact with the resources supported by Docker Cloud. The provider needs to be configured with the proper credentials before it can be used.
---

# Docker Cloud Provider

The Docker Cloud provider is used to interact with the resources
supported by Docker Cloud. The provider needs to be
configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Create variables for user and apikey
variable "dockercloud_user" {}
variable "dockercloud_apikey" {}

# Configure the Docker Cloud Provider
provider "dockercloud" {
    user = "${var.dockercloud_user}"
    apikey = "${var.dockercloud_apikey}"
}

# Create a node cluster
resource "dockercloud_node_cluster" "default" {
    ...
}
```

## Argument Reference

The following arguments are supported:

* `user` - (Required) This is the username for the Docker Cloud account. This
  can also be specified with the `DOCKERCLOUD_USER` shell environment variable.

* `apikey` - (Required) This is the Docker Cloud API key for the user. This
  can also be specified with the `DOCKERCLOUD_APIKEY` shell environment variable.
