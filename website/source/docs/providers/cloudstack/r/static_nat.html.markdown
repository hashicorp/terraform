---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_static_nat"
sidebar_current: "docs-cloudstack-resource-static-nat"
description: |-
  Enables static NAT for a given IP address.
---

# cloudstack\_static\_nat

Enables static NAT for a given IP address

## Example Usage

```
resource "cloudstack_static_nat" "default" {
  ipaddress = "192.168.0.1"
  virtual_machine = "server-1"
}
```

## Argument Reference

The following arguments are supported:

* `ipaddress` - (Required) The name or ID of the public IP address for which
    static NAT will be enabled. Changing this forces a new resource to be
    created.

* `network` - (Optional) The name or ID of the network of the VM the static
    NAT will be enabled for. Required when public IP address is not
    associated with any guest network yet (VPC case). Changing this forces
    a new resource to be created.

* `virtual_machine` - (Required) The name or ID of the virtual machine to
    enable the static NAT feature for. Changing this forces a new resource
    to be created.

* `vm_guest_ip` - (Optional) The virtual machine IP address for the port
    forwarding rule (useful when the virtual machine has a secondairy NIC).
    Changing this forces a new resource to be created.

## Attributes Reference

The following attributes are exported:

* `id` - The static nat ID.
* `network` - The network the public IP address is associated with.
* `vm_guest_ip` - The IP address of the virtual machine that is used
    for the port forwarding rule.
