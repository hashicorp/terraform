---
layout: "aws"
page_title: "AWS: aws_dms_certificate"
sidebar_current: "docs-aws-resource-dms-certificate"
description: |-
  Provides a DMS (Data Migration Service) certificate resource.
---

# aws\_dms\_certificate

Provides a DMS (Data Migration Service) certificate resource. DMS certificates can be created, deleted, and imported.

~> **Note:** All arguments including the PEM encoded certificate will be stored in the raw state as plain-text.
[Read more about sensitive data in state](/docs/state/sensitive-data.html).

## Example Usage

```hcl
# Create a new certificate
resource "aws_dms_certificate" "test" {
  certificate_id  = "test-dms-certificate-tf"
  certificate_pem = "..."
}
```

## Argument Reference

The following arguments are supported:

* `certificate_id` - (Required) The certificate identifier.

    - Must contain from 1 to 255 alphanumeric characters and hyphens.

* `certificate_pem` - (Optional) The contents of the .pem X.509 certificate file for the certificate. Either `certificate_pem` or `certificate_wallet` must be set.
* `certificate_wallet` - (Optional) The contents of the Oracle Wallet certificate for use with SSL. Either `certificate_pem` or `certificate_wallet` must be set.

## Attributes Reference

The following attributes are exported:

* `certificate_arn` - The Amazon Resource Name (ARN) for the certificate.

## Import

Certificates can be imported using the `certificate_arn`, e.g.

```
$ terraform import aws_dms_certificate.test arn:aws:dms:us-west-2:123456789:cert:xxxxxxxxxx
```
