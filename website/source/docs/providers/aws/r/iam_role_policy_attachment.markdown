---
layout: "aws"
page_title: "AWS: aws_iam_role_policy_attachment"
sidebar_current: "docs-aws-resource-iam-role-policy-attachment"
description: |-
  Attaches Managed IAM Policies to an IAM role
---

# aws\_iam\_role\_policy\_attachment

Attaches Managed IAM Policies to an IAM role

```
resource "aws_iam_role" "role" {
    name = "test-role"
}

resource "aws_iam_policy" "policy" {
    name = "test-policy"
    description = "A test policy"
    policy = 	#omitted
}

resource "aws_iam_role_policy_attachment" "test-attach" {
    role = "${aws_iam_role.role.name}"
    policy_arns = ["${aws_iam_policy.policy.arn}"]
}
```

## Argument Reference

The following arguments are supported:

* `role`		(Required) - The role the policy should be applied to
* `policy_arns`	(Required) - A list of ARNs of the policies you want to apply
