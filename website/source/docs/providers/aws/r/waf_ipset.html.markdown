---
layout: "aws"
page_title: "AWS: waf_ipset"
sidebar_current: "docs-aws-resource-waf-ipset"
description: |-
  Provides a AWS WAF IPSet resource.
---

# aws\_waf\_ipset

Provides a WAF IPSet Resource

## Example Usage

```hcl
resource "aws_waf_ipset" "ipset" {
  name = "tfIPSet"

  ip_set_descriptors {
    type  = "IPV4"
    value = "192.0.7.0/24"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name or description of the IPSet.
* `ip_set_descriptors` - (Optional) Specifies the IP address type (IPV4 or IPV6)
	and the IP address range (in CIDR format) that web requests originate from.

## Nested Blocks

### `ip_set_descriptors`

#### Arguments

* `type` - (Required) Type of the IP address - `IPV4` or `IPV6`.
* `value` - (Required) An IPv4 or IPv6 address specified via CIDR notation.
	e.g. `192.0.2.44/32` or `1111:0000:0000:0000:0000:0000:0000:0000/64`

## Remarks

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the WAF IPSet.
