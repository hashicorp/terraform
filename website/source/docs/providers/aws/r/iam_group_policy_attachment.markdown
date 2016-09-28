---
layout: "aws"
page_title: "AWS: aws_iam_group_policy_attachment"
sidebar_current: "docs-aws-resource-iam-group-policy-attachment"
description: |-
  Attaches Managed IAM Policies to an IAM group
---

# aws\_iam\_group\_policy\_attachment

Attaches Managed IAM Policies to an IAM group

```
resource "aws_iam_group" "group" {
    name = "test-group"
}

resource "aws_iam_policy" "policy" {
    name = "test-policy"
    description = "A test policy"
    policy = 	#omitted
}

resource "aws_iam_group_policy_attachment" "test-attach" {
    group = "${aws_iam_group.group.name}"
    policy_arns = ["${aws_iam_policy.policy.arn}"]
}
```

## Argument Reference

The following arguments are supported:

* `group`		(Required) - The group the policy should be applied to
* `policy_arns`	(Required) - A list of ARNs of the policies you want to apply
