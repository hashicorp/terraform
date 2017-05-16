---
layout: "aws"
page_title: "AWS: aws_ebs_snapshot"
sidebar_current: "docs-aws-resource-ebs-snapshot"
description: |-
  Provides an elastic block storage snapshot resource.
---

# aws\_ebs\_snapshot

Creates a Snapshot of an EBS Volume.

## Example Usage

```hcl
resource "aws_ebs_volume" "example" {
    availability_zone = "us-west-2a"
    size = 40
    tags {
        Name = "HelloWorld"
    }
}

resource "aws_ebs_snapshot" "example_snapshot" {
	volume_id = "${aws_ebs_volume.example.id}"
}
```

## Argument Reference

The following arguments are supported:

* `volume_id` - (Required) The Volume ID of which to make a snapshot.
* `description` - (Optional) A description of what the snapshot is.


## Attributes Reference

The following attributes are exported:

* `id` - The snapshot ID (e.g. snap-59fcb34e).
* `owner_id` - The AWS account ID of the EBS snapshot owner.
* `owner_alias` - Value from an Amazon-maintained list (`amazon`, `aws-marketplace`, `microsoft`) of snapshot owners.
* `encrypted` - Whether the snapshot is encrypted.
* `volume_size` - The size of the drive in GiBs.
* `kms_key_id` - The ARN for the KMS encryption key.
* `data_encryption_key_id` - The data encryption key identifier for the snapshot.
* `tags` - A mapping of tags for the resource.