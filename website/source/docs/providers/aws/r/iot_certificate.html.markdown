---
layout: "aws"
page_title: "AWS: aws_iot_certificate"
sidebar_current: "docs-aws-resource-iot-certificate"
description: |-
    Creates and manages an AWS IoT certificate
---

# aws\_iot\_certificate

## Example Usage

```
resource "aws_iot_certificate" "cert" {
  csr = "${file("/my/csr.pem")}"
  active = true
}
```

## Argument Reference

* `csr` - The CSR of the certificate
* `active` - Boolean flag to indicate if the certificate should be active



## Attributes Reference

* `arn` - The ARN of the created AWS IoT certificate
