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

* `name` - (Optional, Forces new resource) The name of the role.
* `name_prefix` - (Optional, Forces new resource) Creates a unique name beginning with the specified prefix. Conflicts with `name`.
* `assume_role_policy` - (Required) The policy that grants an entity permission to assume the role.
* `path` - (Optional) The path to the role.
  See [IAM Identifiers](https://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html) for more information.

## Attributes Reference

The following attributes are exported:

* `arn` - The Amazon Resource Name (ARN) specifying the role.
* `create_date` - The creation date of the IAM role.
* `unique_id` - The stable and unique string identifying the role.

## Import

IAM Roles can be imported using the `name`, e.g.

```
$ terraform import aws_iam_role.developer developer_name
```