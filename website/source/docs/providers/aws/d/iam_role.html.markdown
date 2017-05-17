---
layout: "aws"
page_title: "AWS: aws_iam_role"
sidebar_current: "docs-aws-datasource-iam-role"
description: |-
  Get information on a Amazon IAM role
---

# aws_iam_role

This data source can be used to fetch information about a specific
IAM role. By using this data source, you can reference IAM role
properties without having to hard code ARNs as input.

## Example Usage

```hcl
data "aws_iam_role" "example" {
  role_name = "an_example_role_name"
}
```

## Argument Reference

* `role_name` - (Required) The friendly IAM role name to match.

## Attributes Reference

* `arn` - The Amazon Resource Name (ARN) specifying the role.

* `assume_role_policy_document` - The policy document associated with the role.

* `path` - The path to the role.

* `role_id` - The stable and unique string identifying the role.
