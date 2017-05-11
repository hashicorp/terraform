---
layout: "aws"
page_title: "AWS: ses_domain_identity"
sidebar_current: "docs-aws-resource-ses-domain-identity"
description: |-
  Provides an SES domain identity resource
---

# aws\_ses\_domain_identity

Provides an SES domain identity resource

## Argument Reference

The following arguments are supported:

* `domain` - (Required) The domain name to assign to SES

## Attributes Reference

The following attributes are exported:

* `arn` - The ARN of the domain identity.

* `verification_token` - A code which when added to the domain as a TXT record
  will signal to SES that the owner of the domain has authorised SES to act on
  their behalf. The domain identity will be in state "verification pending"
  until this is done. See below for an example of how this might be achieved
  when the domain is hosted in Route 53 and managed by Terraform.  Find out
  more about verifying domains in Amazon SES in the [AWS SES
  docs](http://docs.aws.amazon.com/ses/latest/DeveloperGuide/verify-domains.html).

## Example Usage

```hcl
resource "aws_ses_domain_identity" "example" {
  domain = "example.com"
}

resource "aws_route53_record" "example_amazonses_verification_record" {
  zone_id = "ABCDEFGHIJ123"
  name    = "_amazonses.example.com"
  type    = "TXT"
  ttl     = "600"
  records = ["${aws_ses_domain_identity.example.verification_token}"]
}
```

