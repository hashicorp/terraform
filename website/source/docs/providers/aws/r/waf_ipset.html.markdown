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

```
resource "aws_waf_ipset" "ipset" {
  name = "tfIPSet"
  ip_set_descriptors {
    type = "IPV4"
    value = "192.0.7.0/24"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name or description of the IPSet.
* `ip_set_descriptors` - (Required) The IP address type and IP address range (in CIDR notation) from which web requests originate. 

## Remarks

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the WAF IPSet.
