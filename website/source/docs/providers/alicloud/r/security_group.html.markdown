---
layout: "alicloud"
page_title: "Alicloud: alicloud_security_group"
sidebar_current: "docs-alicloud-resource-security-group"
description: |-
  Provides a Alicloud Security Group resource.
---

# alicloud\_security\_group

Provides a security group resource.

~> **NOTE:** `alicloud_security_group` is used to build and manage a security group, and `alicloud_security_group_rule` can define ingress or egress rules for it.

## Example Usage

Basic Usage

```
resource "alicloud_security_group" "group" {
  name        = "terraform-test-group"
  description = "New security group"
}
```
Basic usage for vpc

```
resource "alicloud_security_group" "group" {
  name   = "new-group"
  vpc_id = "${alicloud_vpc.vpc.id}"
}

resource "alicloud_vpc" "vpc" {
  cidr_block = "10.1.0.0/21"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) The name of the security group. Defaults to null.
* `description` - (Optional, Forces new resource) The security group description. Defaults to null.
* `vpc_id` - (Optional, Forces new resource) The VPC ID.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the security group
* `vpc_id` - The VPC ID.
* `name` - The name of the security group
* `description` - The description of the security group