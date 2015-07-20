---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_secondary_ipaddress"
sidebar_current: "docs-cloudstack-resource-secondary-ipaddress"
description: |-
  Assigns a secondary IP to a NIC.
---

# cloudstack\_secondary\_ipaddress

Assigns a secondary IP to a NIC.

## Example Usage

```
resource "cloudstack_secondary_ipaddress" "default" {
	virtual_machine = "server-1"
}
```

## Argument Reference

The following arguments are supported:

* `ipaddress` - (Optional) The IP address to attach the to NIC. If not supplied
 		an IP address will be selected randomly. Changing this forces a new resource
		to be	created.

* `nicid` - (Optional) The ID of the NIC to which you want to attach the
		secondary IP address. Changing this forces a new resource to be
    created (defaults to the ID of the primary NIC)

* `virtual_machine` - (Required) The name or ID of the virtual machine to which
 		you want to attach the secondary IP address. Changing this forces a new
		resource to be created.

## Attributes Reference

The following attributes are exported:

* `id` - The secondary IP address ID.
