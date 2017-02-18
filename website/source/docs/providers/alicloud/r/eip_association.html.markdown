---
layout: "alicloud"
page_title: "Alicloud: alicloud_eip_association"
sidebar_current: "docs-alicloud-resource-eip-association"
description: |-
  Provides a ECS EIP Association resource.
---

# alicloud\_eip\_association

Provides an Alicloud EIP Association resource, to associate and disassociate Elastic IPs from ECS Instances.

~> **NOTE:** `alicloud_eip_association` is useful in scenarios where EIPs are either
 pre-existing or distributed to customers or users and therefore cannot be changed.
 In addition, it only supports ECS-VPC.

## Example Usage

```
# Create a new EIP association and use it to associate a EIP form a instance.

resource "alicloud_vpc" "vpc" {
  cidr_block = "10.1.0.0/21"
}

resource "alicloud_vswitch" "vsw" {
  vpc_id            = "${alicloud_vpc.vpc.id}"
  cidr_block        = "10.1.1.0/24"
  availability_zone = "cn-beijing-a"

  depends_on = [
    "alicloud_vpc.vpc",
  ]
}

resource "alicloud_instance" "ecs_instance" {
  image_id              = "ubuntu_140405_64_40G_cloudinit_20161115.vhd"
  instance_type         = "ecs.s1.small"
  availability_zone     = "cn-beijing-a"
  security_groups       = ["${alicloud_security_group.group.id}"]
  vswitch_id            = "${alicloud_vswitch.vsw.id}"
  instance_name         = "hello"
  instance_network_type = "vpc"

  tags {
    Name = "TerraformTest-instance"
  }
}

resource "alicloud_eip" "eip" {}

resource "alicloud_eip_association" "eip_asso" {
  allocation_id = "${alicloud_eip.eip.id}"
  instance_id   = "${alicloud_instance.ecs_instance.id}"
}

resource "alicloud_security_group" "group" {
  name        = "terraform-test-group"
  description = "New security group"
  vpc_id      = "${alicloud_vpc.vpc.id}"
}
```

## Argument Reference

The following arguments are supported:

* `allocation_id` - (Optional, Forces new resource) The allocation EIP ID.
* `instance_id` - (Optional, Forces new resource) The ID of the instance.

## Attributes Reference

The following attributes are exported:

* `allocation_id` - As above.
* `instance_id` - As above.