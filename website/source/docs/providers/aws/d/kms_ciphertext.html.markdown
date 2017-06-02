---
layout: "aws"
page_title: "AWS: aws_kms_ciphertext"
sidebar_current: "docs-aws-datasource-kms-ciphertext"
description: |-
    Provides ciphertext encrypted using a KMS key
---

# aws\_kms\_ciphertext

The KMS ciphertext data source allows you to encrypt plaintext into ciphertext
by using an AWS KMS customer master key.

~> **Note:** All arguments including the plaintext be stored in the raw state as plain-text.
[Read more about sensitive data in state](/docs/state/sensitive-data.html).

## Example Usage

```hcl
resource "aws_kms_key" "oauth_config" {
  description = "oauth config"
  is_enabled = true
}

data "aws_kms_ciphertext" "oauth" {
  key_id = "${aws_kms_key.oauth_config.key_id}"
  plaintext = <<EOF
{
  "client_id": "e587dbae22222f55da22",
  "client_secret": "8289575d00000ace55e1815ec13673955721b8a5"
}
EOF
}
```

## Argument Reference

The following arguments are supported:

* `plaintext` - (Required) Data to be encrypted. Note that this may show up in logs, and it will be stored in the state file.
* `key_id` - (Required) Globally unique key ID for the customer master key.
* `context` - (Optional) An optional mapping that makes up the encryption context.

## Attributes Reference

All of the argument attributes are also exported as result attributes.

* `ciphertext_blob` - Base64 encoded ciphertext
