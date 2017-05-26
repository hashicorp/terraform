---
layout: "aws"
page_title: "AWS: aws_iam_user_policy"
sidebar_current: "docs-aws-resource-iam-user-policy"
description: |-
  Provides an IAM policy attached to a user.
---

# aws\_iam\_user\_policy

Provides an IAM policy attached to a user.

## Example Usage

```hcl
resource "aws_iam_user_policy" "lb_ro" {
  name = "test"
  user = "${aws_iam_user.lb.name}"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "ec2:Describe*"
      ],
      "Effect": "Allow",
      "Resource": "*"
    }
  ]
}
EOF
}

resource "aws_iam_user" "lb" {
  name = "loadbalancer"
  path = "/system/"
}

resource "aws_iam_access_key" "lb" {
  user = "${aws_iam_user.lb.name}"
}
```

## Argument Reference

The following arguments are supported:

* `policy` - (Required) The policy document. This is a JSON formatted string.
	The heredoc syntax or `file` function is helpful here.
* `name` - (Optional) The name of the policy. If omitted, Terraform will assign a random, unique name.
* `name_prefix` - (Optional, Forces new resource) Creates a unique name beginning with the specified prefix. Conflicts with `name`.
* `user` - (Required) IAM user to which to attach this policy.

## Attributes Reference

This resource has no attributes.
