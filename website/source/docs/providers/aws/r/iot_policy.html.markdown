---
layout: "aws"
page_title: "AWS: aws_iot_policy"
sidebar_current: "docs-aws-resource-iot-policy"
description: |-
    Creates and manages an AWS IoT policy
---

# aws\_iot\_policy

## Example Usage

```
resource "aws_iot_policy" "pubsub" {
  name = "PubSubToAnyTopic"
  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": ["iot:*"],
    "Resource": ["*"]
  }]
}
EOF
}
```

## Argument Reference

* `name` - Name of the Policy
* `policy` - Policy document

## Attributes Reference
