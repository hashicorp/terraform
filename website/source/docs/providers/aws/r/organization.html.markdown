---
layout: "aws"
page_title: "AWS: aws_organization
sidebar_current: "docs-aws-resource-organization|"
description: |-
  Provides a resource to create an organization.
---

# aws\_organization

Provides a resource to create an organization.

## Example Usage:

```hcl
resource "aws_organization" "org" {
  feature_set = "ALL"
}
```

## Argument Reference

The following arguments are supported:

* `feature_set` - (Optional) Specify "ALL" (default) or "CONSOLIDATED_BILLING.

## Import

The AWS organization can be imported by using the `account_id`, e.g.

```
$ terraform import aws_organization.my_org 111111111111
```
