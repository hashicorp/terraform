---
layout: "sdc"
page_title: "Smart Data Center: sdc_instance"
sidebar_current: "docs-sdc-resource-instance"
description: |-
  Provides a Smart Data Center/CloudAPI instance resource. This can be used to create, modify, and delete instances. Instances also support provisioning.
---

# sdc\_instance

Provides a Smart Data Center/CloudAPI instance resource. This can be used to
create, modify, and delete instances. Instances also support
[provisioning](/docs/provisioners/index.html).

## Example Usage

```
# Create a new instance
resource "sdc_instance" "foo" {
  name = "bar"
  image = "d34c301e-10c3-11e4-9b79-5f67ca448df0"
  package = "g3-standard-1.75-smartos"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) The instance name.
* `image` - (Required) The instance image UUID.
* `package` - (Required) The instance package.
* `network` - (Required) The list of networks to attach to the resource.
  See [Networks](#networks) below for details.
* `metadata` - (Optional) A mapping of metadata to assign to the resource.
* `tags` - (Optional) A mapping of tags to assign to the resource.

<a id="networks"></a>
## Networks

The `network` block controls which networks should be attached to the instance.
It might help to be familiar with
[SDC Networks](https://docs.joyent.com/private-cloud/networks) to understand
their attributes and naming.

Each `network` supports the following:

* `source` - (Required) The UUID of the network.
* `name` - (Optional) The name to give this interface.

Each `network` will export the following attributes:

* `source` - The attached network
* `name` - The interface name
* `public` - A boolean to mark this network as public
* `address` - The IP attached to the interface

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the instance
* `name`- The name of the instance
* `image` - The image of the instance
* `package` - The package of the instance
* `network` - List of attached networks
* `metadata` - Metadata of the instance
* `tags` - Tags attached to the instance
* `primary_ip` - The primary IPv4 address
* `state` - The instance state (running, starting, stopping, stopped, ...)
* `type` - The type of the instance
* `memory` - The amount of memory the instance has available
* `disk` - The size of the attached data disk
* `created` - Timestamp of instance creation
* `updated` - Timestamp of last instance update

