---
layout: "aws"
page_title: "AWS: aws_iam_group_membership"
sidebar_current: "docs-aws-resource-iam-group-membership"
description: |-
  Provides a top level resource to manage IAM Group membership for IAM Users.
---

# aws\_iam\_group\_membership

Provides a top level resource to manage IAM Group membership for IAM Users. For
more information on managing IAM Groups or IAM Users, see [IAM Groups][1] or
[IAM Users][2]

## Example Usage

```hcl
resource "aws_iam_group_membership" "team" {
  name = "tf-testing-group-membership"

  users = [
    "${aws_iam_user.user_one.name}",
    "${aws_iam_user.user_two.name}",
  ]

  group = "${aws_iam_group.group.name}"
}

resource "aws_iam_group" "group" {
  name = "test-group"
}

resource "aws_iam_user" "user_one" {
  name = "test-user"
}

resource "aws_iam_user" "user_two" {
  name = "test-user-two"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name to identify the Group Membership
* `users` - (Required) A list of IAM User names to associate with the Group
* `group` – (Required) The IAM Group name to attach the list of `users` to

## Attributes Reference

* `name` - The name to identifing the Group Membership
* `users` - list of IAM User names
* `group` – IAM Group name


[1]: /docs/providers/aws/r/iam_group.html
[2]: /docs/providers/aws/r/iam_user.html
