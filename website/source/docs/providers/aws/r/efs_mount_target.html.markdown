---
layout: "aws"
page_title: "AWS: aws_efs_mount_target"
sidebar_current: "docs-aws-resource-efs-mount-target"
description: |-
  Provides an EFS mount target.
---

# aws\_efs\_mount\_target

Provides an EFS mount target. Per [documentation](https://docs.aws.amazon.com/efs/latest/ug/limits.html)
the limit is 1 mount target per AZ for a single EFS file system.

## Example Usage

```
resource "aws_efs_mount_target" "alpha" {
  file_system_id = "${aws_efs_file_system.foo.id}"
  subnet_id = "${aws_subnet.alpha.id}"
}

resource "aws_vpc" "foo" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "alpha" {
  vpc_id = "${aws_vpc.foo.id}"
  availability_zone = "us-west-2a"
  cidr_block = "10.0.1.0/24"
}
```

## Argument Reference

The following arguments are supported:

* `file_system_id` - (Required) The ID of the file system for which the mount target is intended.
* `subnet_id` - (Required) The ID of the subnet that the mount target is in.
* `ip_address` - (Optional) The address at which the file system may be mounted via the mount target.
* `security_groups` - (Optional) A list of up to 5 VPC security group IDs in effect for the mount target.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the mount target
* `network_interface_id` - The ID of the network interface that Amazon EFS created when it created the mount target.
