---
layout: "aws"
page_title: "AWS: aws_iam_policy"
sidebar_current: "docs-aws-resource-iam-policy"
description: |-
  Provides an IAM policy.
---

# aws\_iam\_policy

Provides an IAM policy.

```
resource "aws_iam_policy" "policy" {
    name = "test_policy"
    path = "/"
    description = "My test policy"
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

* `description` - (Optional) Description of the IAM policy.
* `path` - (Optional, default "/") Path in which to create the policy.
* `policy` - (Required) The policy document. This is a JSON formatted string.
  The heredoc syntax or `file` function is helpful here.
* `name` (Required) - The name of the policy.

## Attributes Reference

The following attributes are exported:

* `id` - The policy's ID.
* `arn` - The ARN assigned by AWS to this policy.
* `description` - The description of the policy.
* `name` - The name of the policy.
* `path` - The path of the policy in IAM.
* `policy` - The policy document.
