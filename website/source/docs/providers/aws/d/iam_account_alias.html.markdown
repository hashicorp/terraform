---
layout: "aws"
page_title: "AWS: aws_iam_account_alias"
sidebar_current: "docs-aws-datasource-iam-account-alias"
description: |-
  Provides the account alias for the AWS account associated with the provider
  connection to AWS.
---

# aws\_iam\_account\_alias

The IAM Account Alias data source allows access to the account alias
for the effective account in which Terraform is working.

## Example Usage

```hcl
data "aws_iam_account_alias" "current" {}

output "account_id" {
  value = "${data.aws_iam_account_alias.current.account_alias}"
}
```

## Argument Reference

There are no arguments available for this data source.

## Attributes Reference

The following attributes are exported:

* `account_alias` - The alias associated with the AWS account.
