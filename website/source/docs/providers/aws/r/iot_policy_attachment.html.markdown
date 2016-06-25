---
layout: "aws"
page_title: "AWS: aws_iot_policy_attachment"
sidebar_current: "docs-aws-resource-iot-policy-attachment"
description: |-
    Creates and manages an AWS IoT policy_attachment
---

# aws\_iot\_policy_attachment

## Example Usage

```
resource "aws_iot_policy_attachment" "cert_policies" {
  principal = "${aws_iot_certificate.cert.arn}"
  policies = ["${aws_iot_policy.pubsub.name}"]
}
```

## Argument Reference

* `principal` - The principal of the attachment
* `policies` - List of policies to attach to the principal

## Attributes Reference
