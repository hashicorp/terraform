---
layout: "aws"
page_title: "AWS: aws_iam_role"
sidebar_current: "docs-aws-resource-iam-role"
description: |-
  Provides an IAM role.
---

# aws\_iam\_role

Provides an IAM role.

## Example Usage

```
resource "aws_iam_role" "test_role" {
    name = "test_role"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the role.
* `assume_role_policy` - (Required) The policy that grants an entity permission to assume the role.
* `path` - (Optional) The path to the role.
  See [IAM Identifiers](https://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html) for more information.

## Attributes Reference

* `arn` - The Amazon Resource Name (ARN) specifying the role.
* `unique_id` - The stable and unique string identifying the role.
