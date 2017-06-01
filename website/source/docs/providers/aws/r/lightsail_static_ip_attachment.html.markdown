---
layout: "aws"
page_title: "AWS: aws_lightsail_static_ip_attachment"
sidebar_current: "docs-aws-resource-lightsail-static-ip-attachment"
description: |-
  Provides an Lightsail Static IP Attachment
---

# aws\_lightsail\_static\_ip\_attachment

Provides a static IP address attachment - relationship between a Lightsail static IP & Lightsail instance.

~> **Note:** Lightsail is currently only supported in a limited number of AWS Regions, please see ["Regions and Availability Zones in Amazon Lightsail"](https://lightsail.aws.amazon.com/ls/docs/overview/article/understanding-regions-and-availability-zones-in-amazon-lightsail) for more details

## Example Usage

```hcl
resource "aws_lightsail_static_ip_attachment" "test" {
  static_ip_name = "${aws_lightsail_static_ip.test.name}"
  instance_name = "${aws_lightsail_instance.test.name}"
}

resource "aws_lightsail_static_ip" "test" {
  name = "example"
}

resource "aws_lightsail_instance" "test" {
  name              = "example"
  availability_zone = "us-east-1b"
  blueprint_id      = "string"
  bundle_id         = "string"
  key_pair_name     = "some_key_name"
}
```

## Argument Reference

The following arguments are supported:

* `static_ip_name` - (Required) The name of the allocated static IP
* `instance_name` - (Required) The name of the Lightsail instance to attach the IP to

## Attributes Reference

The following attributes are exported in addition to the arguments listed above:

* `arn` - The ARN of the Lightsail static IP
* `ip_address` - The allocated static IP address
* `support_code` - The support code.
