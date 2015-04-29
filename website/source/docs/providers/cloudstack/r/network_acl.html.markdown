---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_network_acl"
sidebar_current: "docs-cloudstack-resource-network-acl"
description: |-
  Creates a Network ACL for the given VPC.
---

# cloudstack\_network\_acl

Creates a Network ACL for the given VPC.

## Example Usage

```
resource "cloudstack_network_acl" "default" {
	name = "test-acl"
	vpc = "vpc-1"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the ACL. Changing this forces a new resource
    to be created.
* `description` - (Optional) The description of the ACL. Changing this forces a
    new resource to be created.
* `vpc` - (Required) The name or ID of the VPC to create this ACL for. Changing
    this forces a new resource to be created.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the Network ACL

