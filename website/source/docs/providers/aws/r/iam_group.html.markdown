---
layout: "aws"
page_title: "AWS: aws_iam_group"
sidebar_current: "docs-aws-resource-iam-group"
description: |-
  Provides an IAM group.
---

# aws\_iam\_group

Provides an IAM group.

## Example Usage

```
resource "aws_iam_group" "developers" {
    name = "developers"
    path = "/users/"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The group's name.
* `path` - (Optional, default "/") Path in which to create the group.

## Attributes Reference

The following attributes are exported:

* `id` - The group's ID.
* `arn` - The ARN assigned by AWS for this group.
* `name` - The group's name.
* `path` - The path of the group in IAM.
* `unique_id` - The [unique ID][1] assigned by AWS.

  [1]: https://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html#GUIDs
