---
layout: "aws"
page_title: "AWS: aws_kms_alias"
sidebar_current: "docs-aws-datasource-kms-alias"
description: |-
  Get information on a AWS Key Management Service (KMS) Alias
---

# aws\_kms\_alias

Use this data source to get the ARN of a KMS key alias.
By using this data source, you can reference key alias
without having to hard code the ARN as input.

## Example Usage

```hcl
data "aws_kms_alias" "s3" {
  name = "alias/aws/s3"
}
```

## Argument Reference

* `name` - (Required) The display name of the alias. The name must start with the word "alias" followed by a forward slash (alias/)

## Attributes Reference

* `arn` - The Amazon Resource Name(ARN) of the key alias.
* `target_key_id` - Key identifier pointed to by the alias.
