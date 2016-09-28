---
layout: "aws"
page_title: "AWS: aws_iam_user_policy_attachment"
sidebar_current: "docs-aws-resource-iam-user-policy-attachment"
description: |-
  Attaches Managed IAM Policies to an IAM user
---

# aws\_iam\_user\_policy\_attachment

Attaches Managed IAM Policies to an IAM user

```
resource "aws_iam_user" "user" {
    name = "test-user"
}

resource "aws_iam_policy" "policy" {
    name = "test-policy"
    description = "A test policy"
    policy = 	#omitted
}

resource "aws_iam_user_policy_attachment" "test-attach" {
    user = "${aws_iam_user.user.name}"
    policy_arns = ["${aws_iam_policy.policy.arn}"]
}
```

## Argument Reference

The following arguments are supported:

* `user`		(Required) - The user the policy should be applied to
* `policy_arns`	(Required) - A list of ARNs of the policies you want to apply
