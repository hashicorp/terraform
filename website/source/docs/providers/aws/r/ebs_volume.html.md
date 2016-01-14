---
layout: "aws"
page_title: "AWS: aws_ebs_volume"
sidebar_current: "docs-aws-resource-ebs-volume"
description: |-
  Provides an elastic block storage resource.
---

# aws\_ebs\_volume

Manages a single EBS volume.

## Example Usage

```
resource "aws_ebs_volume" "example" {
    availability_zone = "us-west-2a"
    size = 40
    tags {
        Name = "HelloWorld"
    }
}
```

## Argument Reference

The following arguments are supported:

* `availability_zone` - (Required) The AZ where the EBS volume will exist.
* `encrypted` - (Optional) If true, the disk will be encrypted.
* `iops` - (Optional) The amount of IOPS to provision for the disk.
* `size` - (Optional) The size of the drive in GB.
* `snapshot_id` (Optional) A snapshot to base the EBS volume off of.
* `type` - (Optional) The type of EBS volume. Can be "standard", "gp2", or "io1". (Default: "standard").
* `kms_key_id` - (Optional) The KMS key ID for the volume.
* `tags` - (Optional) A mapping of tags to assign to the resource.

## Attributes Reference

The following attributes are exported:

* `id` - The volume ID (e.g. vol-59fcb34e).


