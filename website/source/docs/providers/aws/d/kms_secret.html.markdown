---
layout: "aws"
page_title: "AWS: aws_kms_secret"
sidebar_current: "docs-aws-datasource-kms-secret"
description: |-
    Provides secret data encrypted with the KMS service
---

# aws\_kms\_secret

The KMS secret data source allows you to use data encrypted with the AWS KMS
service within your resource definitions.

~> **NOTE**: Using this data provider will allow you to conceal secret data within your
resource definitions but does not take care of protecting that data in the
logging output, plan output or state output.

Please take care to secure your secret data outside of resource definitions.

## Example Usage

First, let's encrypt a password with KMS using the [AWS CLI
tools](http://docs.aws.amazon.com/cli/latest/reference/kms/encrypt.html).  This
requires you to have your AWS CLI setup correctly, and you would replace the
key-id with your own.

```
$ echo 'master-password' > plaintext-password
$ aws kms encrypt \
> --key-id ab123456-c012-4567-890a-deadbeef123 \
> --plaintext fileb://plaintext-example \
> --encryption-context foo=bar \
> --output text --query CiphertextBlob
AQECAHgaPa0J8WadplGCqqVAr4HNvDaFSQ+NaiwIBhmm6qDSFwAAAGIwYAYJKoZIhvcNAQcGoFMwUQIBADBMBgkqhkiG9w0BBwEwHgYJYIZIAWUDBAEuMBEEDI+LoLdvYv8l41OhAAIBEIAfx49FFJCLeYrkfMfAw6XlnxP23MmDBdqP8dPp28OoAQ==
```

Now, take that output and add it to your resource definitions.

```hcl
data "aws_kms_secret" "db" {
  secret {
    name    = "master_password"
    payload = "AQECAHgaPa0J8WadplGCqqVAr4HNvDaFSQ+NaiwIBhmm6qDSFwAAAGIwYAYJKoZIhvcNAQcGoFMwUQIBADBMBgkqhkiG9w0BBwEwHgYJYIZIAWUDBAEuMBEEDI+LoLdvYv8l41OhAAIBEIAfx49FFJCLeYrkfMfAw6XlnxP23MmDBdqP8dPp28OoAQ=="

    context {
      foo = "bar"
    }
  }
}

resource "aws_rds_cluster" "rds" {
  master_username = "root"
  master_password = "${data.aws_kms_secret.db.master_password}"

  # ...
}
```

And your RDS cluster would have the root password set to "master-password"

## Argument Reference

The following arguments are supported:

* `secret` - (Required) One or more encrypted payload definitions from the KMS
  service.  See the Secret Definitions below.


### Secret Definitions

Each secret definition supports the following arguments:

* `name` - (Required) The name to export this secret under in the attributes.
* `payload` - (Required) Base64 encoded payload, as returned from a KMS encrypt
  opertation.
* `context` - (Optional) An optional mapping that makes up the Encryption
  Context for the secret.
* `grant_tokens` (Optional) An optional list of Grant Tokens for the secret.

For more information on `context` and `grant_tokens` see the [KMS
Concepts](http://docs.aws.amazon.com/kms/latest/developerguide/concepts.html)

## Attributes Reference

Each `secret` defined is exported under its `name` as a top-level attribute.
