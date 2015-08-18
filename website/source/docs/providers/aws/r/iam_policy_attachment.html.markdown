---
layout: "aws"
page_title: "AWS: aws_iam_policy_attachment"
sidebar_current: "docs-aws-resource-iam-policy-attachment"
description: |-
  Attaches a Managed IAM Policy to user(s), role(s), and/or group(s)
---

# aws\_iam\_policy\_attachment

Attaches a Managed IAM Policy to user(s), role(s), and/or group(s)

~> **NOTE:** The aws_iam_policy_attachment resource is only meant to be used once for each managed policy. All of the users/roles/groups that a single policy is being attached to should be declared by a single aws_iam_policy_attachment resource.

```
resource "aws_iam_user" "user" {
    name = "test-user"
}
resource "aws_iam_role" "role" {
    name = "test-role"
}
resource "aws_iam_group" "group" {
    name = "test-group"
}

resource "aws_iam_policy" "policy" {
    name = "test-policy"
    description = "A test policy"
    policy = 	#omitted
}

resource "aws_iam_policy_attachment" "test-attach" {
    name = "test-attachment"
    users = ["${aws_iam_user.user.name}"]
    roles = ["${aws_iam_role.role.name}"]
    groups = ["${aws_iam_group.group.name}"]
    policy_arn = "${aws_iam_policy.policy.arn}"
}
```

## Argument Reference

The following arguments are supported:

* `name` 		(Required) - The name of the policy.
* `users`		(Optional) - The user(s) the policy should be applied to
* `roles`		(Optional) - The role(s) the policy should be applied to
* `groups`		(Optional) - The group(s) the policy should be applied to
* `policy_arn`	(Required) - The ARN of the policy you want to apply

## Attributes Reference

The following attributes are exported:

* `id` - The policy's ID.
* `name` - The name of the policy.
