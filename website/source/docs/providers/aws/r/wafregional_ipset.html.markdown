---
layout: "aws"
page_title: "AWS: wafregional_ipset"
sidebar_current: "docs-aws-resource-wafregional-ipset"
description: |-
  Provides a AWS WAF Regional IPSet resource for use with ALB.
---

# aws\_wafregional\_ipset

Provides a WAF Regional IPSet Resource for use with Application Load Balancer.

## Example Usage

```
resource "aws_wafregional_ipset" "ipset" {
  name = "tfIPSet"
  ip_set_descriptor {
    type = "IPV4"
    value = "192.0.7.0/24"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name or description of the IPSet.
* `ip_set_descriptor` - (Optional) The IP address type and IP address range (in CIDR notation) from which web requests originate.

IPSetDescriptor(ip_set_descriptor) support following:

* `type` - (Required) The string like IPV4 or IPV6.
* `value` - (Required) The CIDR notation.


## Remarks

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the WAF IPSet.
