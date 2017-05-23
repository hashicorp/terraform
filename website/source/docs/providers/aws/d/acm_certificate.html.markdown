---
layout: "aws"
page_title: "AWS: aws_acm_certificate"
sidebar_current: "docs-aws-datasource-acm-certificate"
description: |-
  Get information on a Amazon Certificate Manager (ACM) Certificate
---

# aws\_acm\_certificate

Use this data source to get the ARN of a certificate in AWS Certificate
Manager (ACM). The process of requesting and verifying a certificate in ACM
requires some manual steps, which means that Terraform cannot automate the
creation of ACM certificates. But using this data source, you can reference
them by domain without having to hard code the ARNs as input.

## Example Usage

```hcl
data "aws_acm_certificate" "example" {
  domain   = "tf.example.com"
  statuses = ["ISSUED"]
}
```

## Argument Reference

 * `domain` - (Required) The domain of the certificate to look up. If no certificate is found with this name, an error will be returned.
 * `statuses` - (Optional) A list of statuses on which to filter the returned list. Valid values are `PENDING_VALIDATION`, `ISSUED`,
   `INACTIVE`, `EXPIRED`, `VALIDATION_TIMED_OUT`, `REVOKED` and `FAILED`. If no value is specified, only certificates in the `ISSUED` state
   are returned.

## Attributes Reference

 * `arn` - Set to the ARN of the found certificate, suitable for referencing in other resources that support ACM certificates.
