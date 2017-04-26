---
layout: "aws"
page_title: "AWS: aws_iot_thing"
sidebar_current: "docs-aws-resource-iot-thing"
description: |-
    Creates and manages an AWS IoT thing
---

# aws\_iot\_thing

## Example Usage

```
resource "aws_iot_thing" "device3" {
  name = "MyDevice3"
  principals = ["${aws_iot_certificate.cert.arn}"]

  attributes {
    Manufacturer = "Amazon"
    Type = "IoT Device A"
    SerialNumber = "10293847562912"
  }
}
```

## Argument Reference

* `name` - The name of the IoT thing
* `principals` - List of principals attached to the device
* `attributes` - Map of attributes, i.e. arbitrary key-value pairs that can be attached to a thing


## Attributes Reference

* `id` - The name of the created IoT thing
* `arn` - The ARN of the IoT thing
