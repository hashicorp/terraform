---
layout: "aws"
page_title: "AWS: aws_iam_user"
sidebar_current: "docs-aws-resource-iam-user"
description: |-
  Provides an IAM user.
---

# aws\_iam\_user

Provides an IAM user.

## Example Usage

```hcl
resource "aws_iam_user" "lb" {
  name = "loadbalancer"
  path = "/system/"
}

resource "aws_iam_access_key" "lb" {
  user = "${aws_iam_user.lb.name}"
}

resource "aws_iam_user_policy" "lb_ro" {
  name = "test"
  user = "${aws_iam_user.lb.name}"

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

* `name` - (Required) The user's name. The name must consist of upper and lowercase alphanumeric characters with no spaces. You can also include any of the following characters: `=,.@-_.`. User names are not distinguished by case. For example, you cannot create users named both "TESTUSER" and "testuser".
* `path` - (Optional, default "/") Path in which to create the user.
* `force_destroy` - (Optional, default false) When destroying this user, destroy even if it
  has non-Terraform-managed IAM access keys, login profile or MFA devices. Without `force_destroy`
  a user with non-Terraform-managed access keys and login profile will fail to be destroyed.

## Attributes Reference

The following attributes are exported:

* `arn` - The ARN assigned by AWS for this user.
* `name` - The user's name.
* `unique_id` - The [unique ID][1] assigned by AWS.

  [1]: https://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html#GUIDs


## Import

IAM Users can be imported using the `name`, e.g.

```
$ terraform import aws_iam_user.lb loadbalancer
```
