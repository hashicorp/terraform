---
layout: "aws"
page_title: "AWS: aws_snapshot_create_volume_permission"
sidebar_current: "docs-aws-resource-snapshot-create-volume-permission"
description: |-
  Adds create volume permission to an EBS Snapshot
---

# aws\_snapshot\_create\_volume\_permission

Adds permission to create volumes off of a given EBS Snapshot.

## Example Usage

```hcl
resource "aws_snapshot_create_volume_permission" "example_perm" {
  snapshot_id = "${aws_ebs_snapshot.example_snapshot.id}"
  account_id  = "12345678"
}

resource "aws_ebs_volume" "example" {
  availability_zone = "us-west-2a"
  size              = 40
}

resource "aws_ebs_snapshot" "example_snapshot" {
  volume_id = "${aws_ebs_volume.example.id}"
}
```

## Argument Reference

The following arguments are supported:

  * `snapshot_id` - (required) A snapshot ID
  * `account_id` - (required) An AWS Account ID to add create volume permissions

## Attributes Reference

The following attributes are exported:

  * `id` - A combination of "`snapshot_id`-`account_id`".
