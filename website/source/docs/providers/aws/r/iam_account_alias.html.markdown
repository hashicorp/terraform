---
layout: "aws"
page_title: "AWS: aws_iam_account_alias"
sidebar_current: "docs-aws-resource-iam-account-alias"
description: |-
  Manages the account alias for the AWS Account.
---

# aws\_iam\_account\_alias

-> **Note:** There is only a single account alias per AWS account.

Manages the account alias for the AWS Account.

## Example Usage

```hcl
resource "aws_iam_account_alias" "alias" {
  account_alias = "my-account-alias"
}
```

## Argument Reference

The following arguments are supported:

* `account_alias` - (Required) The account alias

## Import

The current Account Alias can be imported using the `account_alias`, e.g.

```
$ terraform import aws_iam_account_alias.alias my-account-alias
```
