---
layout: "aws"
page_title: "AWS: aws_default_vpc_dhcp_options"
sidebar_current: "docs-aws-resource-default-vpc-dhcp-options"
description: |-
  Manage the default VPC DHCP Options resource.
---

#  aws\_default\_vpc\_dhcp\_options

Provides a resource to manage the [default AWS DHCP Options Set](http://docs.aws.amazon.com/AmazonVPC/latest/UserGuide/VPC_DHCP_Options.html#AmazonDNS)
in the current region.

Each AWS region comes with a default set of DHCP options.
**This is an advanced resource**, and has special caveats to be aware of when
using it. Please read this document in its entirety before using this resource.

The `aws_default_vpc_dhcp_options` behaves differently from normal resources, in that
Terraform does not _create_ this resource, but instead "adopts" it
into management. 

## Example Usage

Basic usage with tags:

```
resource "aws_default_vpc_dhcp_options" "default" {
	tags {
		Name = "Default DHCP Option Set"
	}
}
```

## Argument Reference

The arguments of an `aws_default_vpc_dhcp_options` differ slightly from `aws_vpc_dhcp_options`  resources.
Namely, the `domain_name`, `domain_name_servers` and `ntp_servers` arguments are computed.
The following arguments are still supported: 

* `netbios_name_servers` - (Optional) List of NETBIOS name servers.
* `netbios_node_type` - (Optional) The NetBIOS node type (1, 2, 4, or 8). AWS recommends to specify 2 since broadcast and multicast are not supported in their network. For more information about these node types, see [RFC 2132](http://www.ietf.org/rfc/rfc2132.txt).
* `tags` - (Optional) A mapping of tags to assign to the resource.

### Removing `aws_default_vpc_dhcp_options` from your configuration

The `aws_default_vpc_dhcp_options` resource allows you to manage a region's default DHCP Options Set,
but Terraform cannot destroy it. Removing this resource from your configuration
will remove it from your statefile and management, but will not destroy the DHCP Options Set.
You can resume managing the DHCP Options Set via the AWS Console.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the DHCP Options Set.
