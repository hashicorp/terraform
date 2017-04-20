---
layout: "aws"
page_title: "AWS: aws_iam_user_policy_attachment"
sidebar_current: "docs-aws-resource-iam-user-policy-attachment"
description: |-
  Attaches a Managed IAM Policy to an IAM user
---

# aws\_iam\_user\_policy\_attachment

Attaches a Managed IAM Policy to an IAM user

```hcl
resource "aws_iam_user" "user" {
    name = "test-user"
}

resource "aws_iam_policy" "policy" {
    name        = "test-policy"
    description = "A test policy"
    policy      = # omitted
}

resource "aws_iam_user_policy_attachment" "test-attach" {
    user       = "${aws_iam_user.user.name}"
    policy_arn = "${aws_iam_policy.policy.arn}"
}
```

## Argument Reference

The following arguments are supported:

* `user`		(Required) - The user the policy should be applied to
* `policy_arn`	(Required) - The ARN of the policy you want to apply
