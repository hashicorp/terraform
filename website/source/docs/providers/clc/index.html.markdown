---
layout: "clc"
page_title: "Provider: CenturyLinkCloud"
sidebar_current: "docs-clc-index"
description: |-
  The CenturyLinkCloud provider is used to interact with the many resources
  supported by CLC. The provider needs to be configured with account
  credentials before it can be used.
---

# CLC Provider

The clc provider is used to interact with the many resources supported
by CenturyLinkCloud. The provider needs to be configured with account
credentials before it can be used.

Use the navigation to the left to read about the available resources.

For additional documentation, see the [CLC Developer Center](https://www.ctl.io/developers/)

## Example Usage

```
# Configure the CLC Provider
provider "clc" {
  username = "${var.clc_username}"
  password = "${var.clc_password}"
  account  = "${var.clc_account}" # optional
}

# Create a server
resource "clc_server" "node" {
    ...
}
```


## Account Bootstrap

Trial accounts are available by signing up on the control portal [https://control.ctl.io](https://control.ctl.io).

For new accounts, you should initially run these steps manually:

- [Create a network.](https://control.ctl.io/Network/network)
- [Provision a server.](https://control.ctl.io/create)


## Argument Reference

The following arguments are supported:

* `clc_username` - (Required) This is the CLC account username. It must be provided, but
  it can also be sourced from the `CLC_USERNAME` environment variable.
  
* `clc_password` - (Required) This is the CLC account password. It must be provided, but
  it can also be sourced from the `CLC_PASSWORD` environment variable.
  
* `clc_account` - (Optional) Override CLC account alias. Also taken from the `CLC_ACCOUNT`
  environment variable if provided.
