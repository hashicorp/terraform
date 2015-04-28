---
layout: "aws"
page_title: "AWS: aws_vpc_dhcp_options_association"
sidebar_current: "docs-aws-resource-vpc-dhcp-options-association"
description: |-
  Provides a VPC DHCP Options Association resource.
---

# aws\_vpc\_dhcp\_options\_<wbr>association

Provides a VPC DHCP Options Association resource.

## Example Usage

```
resource "aws_vpc_dhcp_options_association" "dns_resolver" {
	vpc_id = "${aws_vpc.foo.id}"
	dhcp_options_id = "${aws_vpc_dhcp_options.foo.id}"
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC to which we would like to associate a DHCP Options Set.
* `dhcp_options_id` - (Required) The ID of the DHCP Options Set to associate to the VPC.

## Remarks
* You can only associate one DHCP Options Set to a given VPC ID.
* Removing the DHCP Options Association automatically sets AWS's `default` DHCP Options Set to the VPC.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the DHCP Options Set Association.
