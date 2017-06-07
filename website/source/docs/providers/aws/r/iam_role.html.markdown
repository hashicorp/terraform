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

```hcl
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

* `name` - (Optional, Forces new resource) The name of the role. If omitted, Terraform will assign a random, unique name.
* `name_prefix` - (Optional, Forces new resource) Creates a unique name beginning with the specified prefix. Conflicts with `name`.
* `assume_role_policy` - (Required) The policy that grants an entity permission to assume the role.

~> **NOTE:** This `assume_role_policy` is very similar but slightly different than just a standard IAM policy and cannot use an `aws_iam_policy` resource.  It _can_ however, use an `aws_iam_policy_document` [data source](https://www.terraform.io/docs/providers/aws/d/iam_policy_document.html), see example below for how this could work.

* `path` - (Optional) The path to the role.
  See [IAM Identifiers](https://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html) for more information.
* `description` - (Optional) The description of the role.

## Attributes Reference

The following attributes are exported:

* `arn` - The Amazon Resource Name (ARN) specifying the role.
* `create_date` - The creation date of the IAM role.
* `unique_id` - The stable and unique string identifying the role.
* `name` - The name of the role.
* `description` - The description of the role.

## Example of Using Data Source for Assume Role Policy

```hcl
data "aws_iam_policy_document" "instance-assume-role-policy" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "instance" {
  name               = "instance_role"
  path               = "/system/"
  assume_role_policy = "${data.aws_iam_policy_document.instance-assume-role-policy.json}"
}
```

## Import

IAM Roles can be imported using the `name`, e.g.

```
$ terraform import aws_iam_role.developer developer_name
```
