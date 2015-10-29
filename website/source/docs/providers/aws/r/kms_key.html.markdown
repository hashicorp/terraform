---
layout: "aws"
page_title: "AWS: aws_kms_key"
sidebar_current: "docs-aws-resource-kms-key"
description: |-
  Provides a KMS customer master key.
---

# aws\_kms\_key

Provides a KMS customer master key.

## Example Usage

```
resource "aws_kms_key" "a" {
    description = "KMS key 1"
	deletion_window = 10
}
```

## Argument Reference

The following arguments are supported:

* `description` - (Optional) The description of the key as viewed in AWS console.
* `key_usage` - (Optional) Specifies the intended use of the key. Currently this defaults to ENCRYPT/DECRYPT, and only symmetric encryption and decryption are supported.
* `policy` - (Optional) A valid policy JSON document.
* `deletion_window` - (Optional) Duration in days after which the key is deleted after destruction of the resource, must be between 7 and 30 days.

## Attributes Reference

The following attributes are exported:

* `arn` - The Amazon Resource Name (ARN) of the key.
* `key_id` - The globally unique identifier for the key.
