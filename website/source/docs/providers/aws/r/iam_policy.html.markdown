---
layout: "aws"
page_title: "AWS: aws_iam_policy"
sidebar_current: "docs-aws-resource-iam-policy"
description: |-
  Provides an IAM policy.
---

# aws\_iam\_policy

Provides an IAM policy.

```hcl
resource "aws_iam_policy" "policy" {
  name        = "test_policy"
  path        = "/"
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
* `name` - (Optional, Forces new resource) The name of the policy. If omitted, Terraform will assign a random, unique name.
* `name_prefix` - (Optional, Forces new resource) Creates a unique name beginning with the specified prefix. Conflicts with `name`.
* `path` - (Optional, default "/") Path in which to create the policy.
  See [IAM Identifiers](https://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html) for more information.
* `policy` - (Required) The policy document. This is a JSON formatted string.
  The heredoc syntax, `file` function, or the [`aws_iam_policy_document` data
  source](/docs/providers/aws/d/iam_policy_document.html)
  are all helpful here.

## Attributes Reference

The following attributes are exported:

* `id` - The policy's ID.
* `arn` - The ARN assigned by AWS to this policy.
* `description` - The description of the policy.
* `name` - The name of the policy.
* `path` - The path of the policy in IAM.
* `policy` - The policy document.

## Import

IAM Policies can be imported using the `arn`, e.g.

```
$ terraform import aws_iam_policy.administrator arn:aws:iam::123456789012:policy/UsersManageOwnCredentials
```
