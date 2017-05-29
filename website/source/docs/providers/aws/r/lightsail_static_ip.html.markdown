---
layout: "aws"
page_title: "AWS: aws_lightsail_static_ip"
sidebar_current: "docs-aws-resource-lightsail-static-ip"
description: |-
  Provides an Lightsail Static IP
---

# aws\_lightsail\_static\_ip

Allocates a static IP address.

~> **Note:** Lightsail is currently only supported in `us-east-1`, `us-east-2`, `us-west-2`, `eu-west-1`, `eu-west-2`, `eu-central-1` regions.

## Example Usage

```hcl
resource "aws_lightsail_static_ip" "test" {
  name = "example"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name for the allocated static IP

## Attributes Reference

The following attributes are exported in addition to the arguments listed above:

* `arn` - The ARN of the Lightsail static IP
* `ip_address` - The allocated static IP address
* `support_code` - The support code.
