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
  name = "cert_policies"
  principals = ["${aws_iot_certificate.cert.arn}"]
  policy = "${aws_iot_policy.pubsub.name}"
}
```

## Argument Reference

* `name` - A name for the policy attachment
* `principals` - List of principals of the attachment
* `policy` - Policy to attach to the principals

## Attributes Reference
