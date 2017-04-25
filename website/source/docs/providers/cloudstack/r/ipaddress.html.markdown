---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_ipaddress"
sidebar_current: "docs-cloudstack-resource-ipaddress"
description: |-
  Acquires and associates a public IP.
---

# cloudstack_ipaddress

Acquires and associates a public IP.

## Example Usage

```hcl
resource "cloudstack_ipaddress" "default" {
  network_id = "6eb22f91-7454-4107-89f4-36afcdf33021"
}
```

## Argument Reference

The following arguments are supported:

* `network_id` - (Optional) The ID of the network for which an IP address should
    be acquired and associated. Changing this forces a new resource to be created.

* `vpc_id` - (Optional) The ID of the VPC for which an IP address should be
   acquired and associated. Changing this forces a new resource to be created.

* `project` - (Optional) The name or ID of the project to deploy this
    instance to. Changing this forces a new resource to be created.

*NOTE: Either `network_id` or `vpc_id` should have a value!*

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the acquired and associated IP address.
* `ip_address` - The IP address that was acquired and associated.
