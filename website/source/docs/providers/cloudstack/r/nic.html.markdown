---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_nic"
sidebar_current: "docs-cloudstack-resource-nic"
description: |-
  Creates an additional NIC to add a VM to the specified network.
---

# cloudstack\_nic

Creates an additional NIC to add a VM to the specified network.

## Example Usage

Basic usage:

```
resource "cloudstack_nic" "test" {
    network = "network-2"
    ipaddress = "192.168.1.1"
    virtual_machine = "server-1"
}
```

## Argument Reference

The following arguments are supported:

* `network` - (Required) The name or ID of the network to plug the NIC into. Changing
    this forces a new resource to be created.

* `ipaddress` - (Optional) The IP address to assign to the NIC. Changing this
    forces a new resource to be created.

* `virtual_machine` - (Required) The name or ID of the virtual machine to which
    to attach the NIC. Changing this forces a new resource to be created.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the NIC.
* `ipaddress` - The assigned IP address.
