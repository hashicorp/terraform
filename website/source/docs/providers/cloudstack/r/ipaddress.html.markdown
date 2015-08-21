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

* `network` - (Optional) The name or ID of the network for which an IP address should
    be acquired and associated. Changing this forces a new resource to be created.

* `vpc` - (Optional) The name or ID of the VPC for which an IP address should
    be acquired and associated. Changing this forces a new resource to be created.

* `project` - (Optional) The name or ID of the project to deploy this
    instance to. Changing this forces a new resource to be created.

*NOTE: Either `network` or `vpc` should have a value!*

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the acquired and associated IP address.
* `ipaddress` - The IP address that was acquired and associated.
