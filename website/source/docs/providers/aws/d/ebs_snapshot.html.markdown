---
layout: "aws"
page_title: "AWS: aws_ebs_snapshot"
sidebar_current: "docs-aws-datasource-ebs-snapshot"
description: |-
  Get information on an EBS Snapshot.
---

# aws\_ebs\_snapshot

Use this data source to get information about an EBS Snapshot for use when provisioning EBS Volumes

## Example Usage

```hcl
data "aws_ebs_snapshot" "ebs_volume" {
  most_recent = true
  owners      = ["self"]

  filter {
    name   = "volume-size"
    values = ["40"]
  }

  filter {
    name   = "tag:Name"
    values = ["Example"]
  }
}
```

## Argument Reference

The following arguments are supported:

* `most_recent` - (Optional) If more than one result is returned, use the most recent snapshot.

* `owners` - (Optional) Returns the snapshots owned by the specified owner id. Multiple owners can be specified.

* `snapshot_ids` - (Optional) Returns information on a specific snapshot_id.

* `restorable_by_user_ids` - (Optional) One or more AWS accounts IDs that can create volumes from the snapshot.

* `filter` - (Optional) One or more name/value pairs to filter off of. There are
several valid keys, for a full reference, check out
[describe-volumes in the AWS CLI reference][1].


## Attributes Reference

The following attributes are exported:

* `id` - The snapshot ID (e.g. snap-59fcb34e).
* `snapshot_id` - The snapshot ID (e.g. snap-59fcb34e).
* `description` - A description for the snapshot
* `owner_id` - The AWS account ID of the EBS snapshot owner.
* `owner_alias` - Value from an Amazon-maintained list (`amazon`, `aws-marketplace`, `microsoft`) of snapshot owners.
* `volume_id` - The volume ID (e.g. vol-59fcb34e).
* `encrypted` - Whether the snapshot is encrypted.
* `volume_size` - The size of the drive in GiBs.
* `kms_key_id` - The ARN for the KMS encryption key.
* `data_encryption_key_id` - The data encryption key identifier for the snapshot.
* `state` - The snapshot state.
* `tags` - A mapping of tags for the resource.

[1]: http://docs.aws.amazon.com/cli/latest/reference/ec2/describe-snapshots.html
