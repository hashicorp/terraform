---
layout: "tutum"
page_title: "Provider: Tutum"
sidebar_current: "docs-tutum-index"
description: |-
  The Tutum provider is used to interact with the resources supported by the Tutum Cloud. The provider needs to be configured with the proper credentials before it can be used.
---

# Tutum Provider

The Tutum provider is used to interact with the resources
supported by the Tutum Cloud. The provider needs to be
configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Create variables for user and apikey
variable "tutum_user" {}
variable "tutum_apikey" {}

# Configure the Tutum Provider
provider "tutum" {
    user = "${var.tutum_user}"
    apikey = "${var.tutum_apikey}"
}

# Create a node cluster
resource "tutum_node_cluster" "default" {
    ...
}
```

## Argument Reference

The following arguments are supported:

* `user` - (Required) This is the username for the Tutum account. This
  can also be specified with the `TUTUM_USER` shell environment variable.

* `apikey` - (Required) This is the Tutum API key for the user. This
  can also be specified with the `TUTUM_APIKEY` shell environment variable.
