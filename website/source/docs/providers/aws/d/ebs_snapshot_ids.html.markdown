---
layout: "aws"
page_title: "AWS: aws_ebs_snapshot_ids"
sidebar_current: "docs-aws-datasource-ebs-snapshot-ids"
description: |-
  Provides a list of EBS snapshot IDs.
---

# aws\_ebs\_snapshot\_ids

Use this data source to get a list of EBS Snapshot IDs matching the specified
criteria.

## Example Usage

```hcl
data "aws_ebs_snapshot_ids" "ebs_volumes" {
  owners = ["self"]

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

* `owners` - (Optional) Returns the snapshots owned by the specified owner id. Multiple owners can be specified.

* `restorable_by_user_ids` - (Optional) One or more AWS accounts IDs that can create volumes from the snapshot.

* `filter` - (Optional) One or more name/value pairs to filter off of. There are
several valid keys, for a full reference, check out
[describe-volumes in the AWS CLI reference][1].

## Attributes Reference

`ids` is set to the list of EBS snapshot IDs, sorted by creation time in
descending order.

[1]: http://docs.aws.amazon.com/cli/latest/reference/ec2/describe-snapshots.html
