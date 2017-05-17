---
layout: "aws"
page_title: "AWS: aws_caller_identity"
sidebar_current: "docs-aws-datasource-caller-identity"
description: |-
  Get information about the identity of the caller for the provider
  connection to AWS.
---

# aws\_caller\_identity

Use this data source to get the access to the effective Account ID, User ID, and ARN in
which Terraform is authorized.

## Example Usage

```hcl
data "aws_caller_identity" "current" {}

output "account_id" {
  value = "${data.aws_caller_identity.current.account_id}"
}

output "caller_arn" {
  value = "${data.aws_caller_identity.current.arn}"
}

output "caller_user" {
  value = "${data.aws_caller_identity.current.user_id}"
}
```

## Argument Reference

There are no arguments available for this data source.

## Attributes Reference

* `account_id` - The AWS Account ID number of the account that owns or contains the calling entity.
* `arn` - The AWS ARN associated with the calling entity.
* `user_id` - The unique identifier of the calling entity.
