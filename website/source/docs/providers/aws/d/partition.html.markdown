---
layout: "aws"
page_title: "AWS: aws_partition"
sidebar_current: "docs-aws-datasource-partition"
description: |-
  Get AWS partition identifier
---

# aws\_partition

Use this data source to lookup current AWS partition in which Terraform is working

## Example Usage

```hcl
data "aws_partition" "current" {}

data "aws_iam_policy_document" "s3_policy" {
  statement {
    sid = "1"

    actions = [
      "s3:ListBucket",
    ]

    resources = [
      "arn:${data.aws_partition.current.partition}:s3:::my-bucket",
    ]
  }
}
```

## Argument Reference

There are no arguments available for this data source.

## Attributes Reference

`partition` is set to the identifier of the current partition.
