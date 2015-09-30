---
layout: "linode"
page_title: "Provider: Linode"
sidebar_current: "docs-linode-index"
description: |-
  The Linode provider is used to interact with the resources supported by Linode. The provider needs to be configured with the proper credentials before it can be used.
---
# Linode Provider

The Linode provider is used to interact with the resources supported by Linode.
The provider needs to be configured with the proper credentials before it can
be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the Linode Provider
provider "linode" {
  key = "${var.linode_token}"
}

# create a web server
resource "linode_linode" "web" {
  ...
}
```

## Argument Reference

The following arguments are supported:

* `key` - (Required) This is your Linode API key. This can also be specified
  with the `LINODE_API_KEY` environment variable.
