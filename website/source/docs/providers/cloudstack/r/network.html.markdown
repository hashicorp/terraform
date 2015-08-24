---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_network"
sidebar_current: "docs-cloudstack-resource-network"
description: |-
  Creates a network.
---

# cloudstack\_network

Creates a network.

## Example Usage

Basic usage:

```
resource "cloudstack_network" "default" {
    name = "test-network"
    cidr = "10.0.0.0/16"
    network_offering = "Default Network"
    zone = "zone-1"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the network.

* `display_text` - (Optional) The display text of the network.

* `cidr` - (Required) The CIDR block for the network. Changing this forces a new
    resource to be created.

* `network_offering` - (Required) The name or ID of the network offering to use
    for this network.

* `vpc` - (Optional) The name or ID of the VPC to create this network for. Changing
    this forces a new resource to be created.

* `aclid` - (Optional) The ID of a network ACL that should be attached to the
    network. Changing this forces a new resource to be created.

* `project` - (Optional) The name or ID of the project to deploy this
    instance to. Changing this forces a new resource to be created.

* `zone` - (Required) The name or ID of the zone where this disk volume will be
    available. Changing this forces a new resource to be created.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the network.
* `display_text` - The display text of the network.
