---
layout: "aws"
page_title: "AWS: aws_iam_group_policy_attachment"
sidebar_current: "docs-aws-resource-iam-group-policy-attachment"
description: |-
  Attaches a Managed IAM Policy to an IAM group
---

# aws\_iam\_group\_policy\_attachment

Attaches a Managed IAM Policy to an IAM group

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
    name = "test-attachment"
    group = "${aws_iam_group.group.name}"
    policy_arn = "${aws_iam_policy.policy.arn}"
}
```

## Argument Reference

The following arguments are supported:

* `name` 		(Required) - The name of the policy.
* `group`		(Required) - The group the policy should be applied to
* `policy_arn`	(Required) - The ARN of the policy you want to apply

## Attributes Reference

The following attributes are exported:

* `id` - The policy's ID.
* `name` - The name of the policy.
