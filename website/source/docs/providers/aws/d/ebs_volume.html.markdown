---
layout: "aws"
page_title: "AWS: aws_ebs_volume"
sidebar_current: "docs-aws-datasource-ebs-volume"
description: |-
  Get information on an EBS volume.
---

# aws\_ebs\_volume

Use this data source to get information about an EBS volume for use in other
resources.

## Example Usage

```hcl
data "aws_ebs_volume" "ebs_volume" {
  most_recent = true

  filter {
    name   = "volume-type"
    values = ["gp2"]
  }

  filter {
    name   = "tag:Name"
    values = ["Example"]
  }
}
```

## Argument Reference

The following arguments are supported:

* `most_recent` - (Optional) If more than one result is returned, use the most
recent Volume.
* `filter` - (Optional) One or more name/value pairs to filter off of. There are
several valid keys, for a full reference, check out
[describe-volumes in the AWS CLI reference][1].


## Attributes Reference

The following attributes are exported:

* `id` - The volume ID (e.g. vol-59fcb34e).
* `volume_id` - The volume ID (e.g. vol-59fcb34e).
* `availability_zone` - The AZ where the EBS volume exists.
* `encrypted` - Whether the disk is encrypted.
* `iops` - The amount of IOPS for the disk.
* `size` - The size of the drive in GiBs.
* `snapshot_id` - The snapshot_id the EBS volume is based off.
* `volume_type` - The type of EBS volume.
* `kms_key_id` - The ARN for the KMS encryption key.
* `tags` - A mapping of tags for the resource.

[1]: http://docs.aws.amazon.com/cli/latest/reference/ec2/describe-volumes.html
