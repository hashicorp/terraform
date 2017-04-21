---
layout: "aws"
page_title: "AWS: aws_canonical_user_id"
sidebar_current: "docs-aws-datasource-canonical-user-id"
description: |-
  Provides the canonical user ID for the AWS account associated with the provider
  connection to AWS.
---

# aws\_canonical\_user\_id

The Canonical User ID data source allows access to the [canonical user ID](http://docs.aws.amazon.com/general/latest/gr/acct-identifiers.html)
for the effective account in which Terraform is working.

## Example Usage

```hcl
data "aws_canonical_user_id" "current" {}

output "canonical_user_id" {
  value = "${data.aws_canonical_user_id.current.id}"
}
```

## Argument Reference

There are no arguments available for this data source.

## Attributes Reference

The following attributes are exported:

* `id` - The canonical user ID associated with the AWS account.

* `display_name` - The human-friendly name linked to the canonical user ID.
