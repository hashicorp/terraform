---
layout: "aws"
page_title: "AWS: aws_ami_launch_permission"
sidebar_current: "docs-aws-resource-ami-launch-permission"
description: |-
  Adds launch permission to Amazon Machine Image (AMI).
---

# aws\_ami\_launch\_permission

Adds launch permission to Amazon Machine Image (AMI) from another AWS account.

## Example Usage

```hcl
resource "aws_ami_launch_permission" "example" {
  image_id   = "ami-12345678"
  account_id = "123456789012"
}
```

## Argument Reference

The following arguments are supported:

  * `image_id` - (required) A region-unique name for the AMI.
  * `account_id` - (required) An AWS Account ID to add launch permissions.

## Attributes Reference

The following attributes are exported:

  * `id` - A combination of "`image_id`-`account_id`".
