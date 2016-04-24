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

* `startip` - (Optional) Start of the IP block that will be available on the 
    network. Defaults to the second available IP in the range.

* `endip` - (Optional) End of the IP block that will be available on the 
    network. Defaults to the last available IP in the range.

* `gateway` - (Optional) Gateway that will be provided to the instances in this
    network. Defaults to the first usable IP in the range.

* `network_offering` - (Required) The name or ID of the network offering to use
    for this network.

* `vlan` - (Optional) The VLAN number (1-4095) the network will use. This might be
    required by the Network Offering if specifyVlan=true is set. Only the ROOT 
    admin can set this value.

* `vpc_id` - (Optional) The ID of the VPC to create this network for. Changing
    this forces a new resource to be created.

* `vpc` - (Optional, Deprecated) The name or ID of the VPC to create this network
    for. Changing this forces a new resource to be created.

* `acl_id` - (Optional) The network ACL ID that should be attached to the network.
    Changing this forces a new resource to be created.

* `aclid` - (Optional, Deprecated) The ID of a network ACL that should be attached
    to the network. Changing this forces a new resource to be created.

* `project` - (Optional) The name or ID of the project to deploy this
    instance to. Changing this forces a new resource to be created.

* `zone` - (Required) The name or ID of the zone where this network will be
    available. Changing this forces a new resource to be created.

* `tags` - (Optional) A mapping of tags to assign to the resource. 

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the network.
* `display_text` - The display text of the network.
