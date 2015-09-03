---
layout: "aws"
page_title: "AWS: aws_iam_access_key"
sidebar_current: "docs-aws-resource-iam-access-key"
description: |-
  Provides an IAM access key. This is a set of credentials that allow API requests to be made as an IAM user.
---

# aws\_iam\_access\_key

Provides an IAM access key. This is a set of credentials that allow API requests to be made as an IAM user.

## Example Usage

```
resource "aws_iam_access_key" "lb" {
    user = "${aws_iam_user.lb.name}"
}

resource "aws_iam_user" "lb" {
    name = "loadbalancer"
    path = "/system/"
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

* `user` - (Required) The IAM user to associate with this access key.

## Attributes Reference

The following attributes are exported:

* `id` - The access key ID.
* `user` - The IAM user associated with this access key.
* `secret` - The secret access key. Note that this will be written to the state file.
* `ses_smtp_password` - The secret access key converted into an SES SMTP
  password by applying [AWS's documented conversion
  algorithm](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/smtp-credentials.html#smtp-credentials-convert).
* `status` - "Active" or "Inactive". Keys are initially active, but can be made
	inactive by other means.
