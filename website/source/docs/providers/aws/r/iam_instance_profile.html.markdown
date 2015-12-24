---
layout: "aws"
page_title: "AWS: aws_iam_instance_profile"
sidebar_current: "docs-aws-resource-iam-instance-profile"
description: |-
  Provides an IAM instance profile.
---

# aws\_iam\_instance\_profile

Provides an IAM instance profile.

## Example Usage

```
resource "aws_iam_instance_profile" "test_profile" {
    name = "test_profile"
    roles = ["${aws_iam_role.role.name}"]
}

resource "aws_iam_role" "role" {
    name = "test_role"
    path = "/"
    assume_role_policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": "sts:AssumeRole",
            "Principal": {"AWS": "*"},
            "Effect": "Allow",
            "Sid": ""
        }
    ]
}
EOF
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The profile's name.
* `path` - (Optional, default "/") Path in which to create the profile.
* `roles` - (Required) A list of role names to include in the profile.

## Attribute Reference

* `id` - The instance profile's ID.
* `arn` - The ARN assigned by AWS to the instance profile.
* `create_date` - The creation timestamp of the instance profile.
* `name` - The instance profile's name.
* `path` - The path of the instance profile in IAM.
* `roles` - The list of roles assigned to the instance profile.
* `unique_id` - The [unique ID][1] assigned by AWS.

  [1]: http://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html#GUIDs
