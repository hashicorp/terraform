---
layout: "aws"
page_title: "AWS: aws_kms_secrets"
sidebar_current: "docs-aws-datasource-kms-secrets"
description: |-
	Provides secret data encrypted with the KMS service
---

# aws\_kms\_secrets

The KMS secrets data source allows you to use data encrypted with the AWS KMS
service within your resource definitions.

## Note about encrypted data

Using this data provider will allow you to conceal secret data within your
resource definitions but does not take care of protecting that data in the
logging output, plan output or state output.

Please take care to secure your secret data outside of resource definitions.

## Example Usage

```
data "aws_kms_secrets" "db" {
    secret {
        name = "master_password"
        payload = "AQECAHhhE7rnmbnLg..."

        context {
            foo = "bar"
        }
    }
}

resource "aws_rds_cluster" "rds" {
    master_username = "root"
    master_password = "${data.aws_kms_secrets.db.master_password}"
    ...
}
```

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
