---
layout: "vcd"
page_title: "Provider: vCloudDirector"
sidebar_current: "docs-vcd-index"
description: |-
  The vCloud Director provider is used to interact with the resources supported by vCloud
  Director. The provider needs to be configured with the proper credentials before it can be used.
---

# vCloud Director Provider

The vCloud Director provider is used to interact with the resources supported by vCloud
Director. The provider needs to be configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

~> **NOTE:** The vCloud Director Provider currently represents _initial support_ and
therefore may undergo significant changes as the community improves it.

## Example Usage

```
# Configure the vCloud Director Provider
provider "vcd" {
	user     = "${var.vcd_user}"
	password = "${var.vcd_pass}"
	org      = "${var.vcd_org}"
	url      = "${var.vcd_url}"
	vdc      = "${var.vcd_vdc}"
}

# Create a new network
resource "vcd_network" "net" {
    ...
}
```

## Argument Reference

The following arguments are used to configure the vCloud Director Provider:

* `user` - (Required) This is the username for vCloud Director API operations. Can also
  be specified with the `VCD_USER` environment variable.
* `password` - (Required) This is the password for vCloud Director API operations. Can
  also be specified with the `VCD_PASSWORD` environment variable.
* `org` - (Required) This is the vCloud Director Org on which to run API
  operations. Can also be specified with the `VCD_ORG` environment
  variable.
* `url` - (Required) This is the URL for the vCloud Director API.
  Can also be specified with the `VCD_URL` environment variable.
* `vdc` - (Optional) This is the virtual datacenter within vCloud Director to run
  API operations against. If not set the plugin will select the first virtual
  datacenter available to your Org. Can also be specified with the `VCD_VDC` environment
  variable.
