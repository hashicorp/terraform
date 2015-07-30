---
layout: "aws"
page_title: "AWS: aws_iam_user"
sidebar_current: "docs-aws-resource-iam-user"
description: |-
  Provides an IAM user.
---

# aws\_iam\_user

Provides an IAM user.

## Example Usage

```
resource "aws_iam_user" "lb" {
    name = "loadbalancer"
    path = "/system/"
}

resource "aws_iam_access_key" "lb" {
    user = "${aws_iam_user.lb.name}"
}

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
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The user's name.
* `path` - (Optional, default "/") Path in which to create the user.

## Attributes Reference

The following attributes are exported:

* `unique_id` - The [unique ID][1] assigned by AWS.
* `arn` - The ARN assigned by AWS for this user.

  [1]: http://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html#GUIDs
