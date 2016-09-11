---
layout: "aws"
page_title: "AWS: aws_kms_key_encrypt"
sidebar_current: "docs-aws-resource-kms-key-encrypt"
description: |-
  encrypts content for a KMS key
---

# aws\_kms\_key\_encrypt

Encrypts content for an aws_kms_key.

## Example Usage

```
resource "aws_kms_key" "oauth_config" {
  description = "oauth config"
  is_enabled = true
}

resource "aws_kms_key_encrypt" "oauth" {
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

* `key_id` - (Required) the KMS key to be used
* `plaintext` - (Required) the plaintext content

## Attributes Reference

The following attributes are exported:

* `plaintext_blob` - Base64 encoded ciphertext blob
