---
layout: "aws"
page_title: "AWS: aws_opsworks_user_profile"
sidebar_current: "docs-aws-resource-opsworks-user-profile"
description: |-
  Provides an OpsWorks User Profile resource.
---

# aws\_opsworks\_user\_profile

Provides an OpsWorks User Profile resource.

## Example Usage

```hcl
resource "aws_opsworks_user_profile" "my_profile" {
  user_arn     = "${aws_iam_user.user.arn}"
  ssh_username = "my_user"
}
```

## Argument Reference

The following arguments are supported:

* `user_arn` - (Required) The user's IAM ARN
* `allow_self_management` - (Optional) Whether users can specify their own SSH public key through the My Settings page
* `ssh_username` - (Required) The ssh username, with witch this user wants to log in
* `ssh_public_key` - (Optional) The users public key

## Attributes Reference

The following attributes are exported:

* `id` - Same value as `user_arn`
