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
  ip_address_id = "f8141e2f-4e7e-4c63-9362-986c908b7ea7"
  virtual_machine_id = "6ca2a163-bc68-429c-adc8-ab4a620b1bb3"
}
```

## Argument Reference

The following arguments are supported:

* `ip_address_id` - (Required) The public IP address ID for which static
    NAT will be enabled. Changing this forces a new resource to be created.

* `network_id` - (Deprecated) The network ID of the VM the static NAT will be
    enabled for. This argument is no longer needed and can be safely omitted.

* `virtual_machine_id` - (Required) The virtual machine ID to enable the
    static NAT feature for. Changing this forces a new resource to be created.

* `vm_guest_ip` - (Optional) The virtual machine IP address for the port
    forwarding rule (useful when the virtual machine has a secondairy NIC).
    Changing this forces a new resource to be created.

* `project` - (Optional) The name or ID of the project to deploy this
    instance to. Changing this forces a new resource to be created.

## Attributes Reference

The following attributes are exported:

* `id` - The static nat ID.
* `vm_guest_ip` - The IP address of the virtual machine that is used
    for the port forwarding rule.
