---
layout: "aws"
page_title: "AWS: aws_kms_alias"
sidebar_current: "docs-aws-resource-kms-alias"
description: |-
  Provides a display name for a customer master key.
---

# aws\_kms\_alias

Provides an alias for a KMS customer master key. AWS Console enforces 1-to-1 mapping between aliases & keys,
but API (hence Terraform too) allows you to create as many aliases as
the [account limits](http://docs.aws.amazon.com/kms/latest/developerguide/limits.html) allow you.

## Example Usage

```
resource "aws_kms_key" "a" {
}

resource "aws_kms_alias" "a" {
    name = "alias/my-key-alias"
    target_key_id = "${aws_kms_key.a.key_id}"
}
```

## Argument Reference

The following arguments are supported:


* `name` - (Optional) The display name of the alias. The name must start with the word "alias" followed by a forward slash (alias/)
* `name_prefix` - (Optional) Creates an unique alias beginning with the specified prefix.  
The name must start with the word "alias" followed by a forward slash (alias/).  Conflicts with `name`.
* `target_key_id` - (Required) Identifier for the key for which the alias is for, can be either an ARN or key_id.

## Attributes Reference

The following attributes are exported:

* `arn` - The Amazon Resource Name (ARN) of the key alias.
