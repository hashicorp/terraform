---
layout: "aws"
page_title: "AWS: aws_caller_identity"
sidebar_current: "docs-aws-datasource-caller-identity"
description: |-
  Get information about the identity of the caller for the provider
  connection to AWS.
---

# aws\_caller\_identity

Use this data source to get the access to the effective Account ID in
which Terraform is working.

~> **NOTE on `aws_caller_identity`:** - an Account ID is only available
if `skip_requesting_account_id` is not set on the AWS provider. In such
cases, the data source will return an error.

## Example Usage

```
data "aws_caller_identity" "current" { }

output "account_id" {
  value = "${data.aws_caller_identity.current.account_id}"
}
```

## Argument Reference

There are no arguments available for this data source.

## Attributes Reference

`account_id` is set to the ID of the AWS account. 
