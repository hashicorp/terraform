---
layout: "aws"
page_title: "AWS: aws_iam_server_certificate"
sidebar_current: "docs-aws-iam-server-certificate"
description: |-
  Get information about a server certificate
---

# aws\_iam\_server\_certificate

Use this data source to lookup information about IAM Server Certificates.

## Example Usage

```hcl
data "aws_iam_server_certificate" "my-domain" {
  name_prefix = "my-domain.org"
  latest      = true
}

resource "aws_elb" "elb" {
  name = "my-domain-elb"

  listener {
    instance_port      = 8000
    instance_protocol  = "https"
    lb_port            = 443
    lb_protocol        = "https"
    ssl_certificate_id = "${data.aws_iam_server_certificate.my-domain.arn}"
  }
}
```

## Argument Reference

* `name_prefix` - prefix of cert to filter by
* `name` - exact name of the cert to lookup
* `latest` - sort results by expiration date. returns the certificate with expiration date in furthest in the future.

## Attributes Reference

`arn` is set to the ARN of the IAM Server Certificate
`path` is set to the path of the IAM Server Certificate
`expiration_date` is set to the expiration date of the IAM Server Certificate

## Import 

The terraform import function will read in certificate body, certificate chain (if it exists), id, name, path, and arn. 
It will not retrieve the private key which is not available through the AWS API.   

 
