---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_ipaddress"
sidebar_current: "docs-cloudstack-resource-ipaddress"
description: |-
  Acquires and associates a public IP.
---

# cloudstack\_ipaddress

Acquires and associates a public IP.

## Example Usage

```
resource "cloudstack_ipaddress" "default" {
  network = "test-network"
}
```

## Argument Reference

The following arguments are supported:

* `network` - (Optional) The name of the network for which an IP address should
    be aquired and accociated. Changing this forces a new resource to be created.

* `vpc` - (Optional) The name of the vpc for which an IP address should
    be aquired and accociated. Changing this forces a new resource to be created.

*NOTE: Either `network` or `vpc` should have a value!*

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the aquired and accociated IP address.
* `ipaddress` - The IP address that was aquired and accociated.
