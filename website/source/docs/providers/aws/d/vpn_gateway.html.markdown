---
layout: "aws"
page_title: "AWS: aws_vpn_gateway"
sidebar_current: "docs-aws-datasource-vpn-gateway"
description: |-
    Provides details about a specific VPN gateway.
---

# aws\_vpn\_gateway

The VPN Gateway data source provides details about
a specific VPN gateway.

## Example Usage

```hcl
data "aws_vpn_gateway" "selected" {
  filter {
    name = "tag:Name"
    values = ["vpn-gw"]
  }
}

output "vpn_gateway_id" {
  value = "${data.aws_vpn_gateway.selected.id}"
}
```

## Argument Reference

The arguments of this data source act as filters for querying the available VPN gateways.
The given filters must match exactly one VPN gateway whose data will be exported as attributes.

* `id` - (Optional) The ID of the specific VPN Gateway to retrieve.

* `state` - (Optional) The state of the specific VPN Gateway to retrieve.

* `availability_zone` - (Optional) The Availability Zone of the specific VPN Gateway to retrieve.

* `attached_vpc_id` - (Optional) The ID of a VPC attached to the specific VPN Gateway to retrieve.

* `filter` - (Optional) Custom filter block as described below.

* `tags` - (Optional) A mapping of tags, each pair of which must exactly match
  a pair on the desired VPN Gateway.

More complex filters can be expressed using one or more `filter` sub-blocks,
which take the following arguments:

* `name` - (Required) The name of the field to filter by, as defined by
  [the underlying AWS API](http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeVpnGateways.html).

* `values` - (Required) Set of values that are accepted for the given field.
  A VPN Gateway will be selected if any one of the given values matches.

## Attributes Reference

All of the argument attributes are also exported as result attributes.
