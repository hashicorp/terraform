---
layout: "aws"
page_title: "AWS: aws_vpc_dhcp_options"
sidebar_current: "docs-aws-resource-vpc-dhcp-options"
description: |-
  Provides a VPC DHCP Options resource.
---

# aws\_vpc\_dhcp\_options

Provides a VPC DHCP Options resource.

## Example Usage

Basic usage:

```
resource "aws_vpc_dhcp_options" "dns_resolver" {
	domain_name_servers = ["8.8.8.8", "8.8.4.4"]
}
```

Full usage:

```
resource "aws_vpc_dhcp_options" "foo" {
	domain_name = "service.consul"
	domain_name_servers = ["127.0.0.1", "10.0.0.2"]
	ntp_servers = ["127.0.0.1"]
	netbios_name_servers = ["127.0.0.1"]
	netbios_node_type = 2

	tags {
		Name = "foo-name"
	}
}
```

## Argument Reference

The following arguments are supported:

* `domain_name` - (Optional) the suffix domain name to use by default when resolving non Fully Qualified Domain Names. In other words, this is what ends up being the `search` value in the `/etc/resolv.conf` file.
* `domain_name_servers` - (Optional) List of name servers to configure in `/etc/resolv.conf`.
* `ntp_servers` - (Optional) List of NTP servers to configure.
* `netbios_name_servers` - (Optional) List of NETBIOS name servers.
* `netbios_node_type` - (Optional) The NetBIOS node type (1, 2, 4, or 8). AWS recommends to specify 2 since broadcast and multicast are not supported in their network. For more information about these node types, see [RFC 2132](http://www.ietf.org/rfc/rfc2132.txt).
* `tags` - (Optional) A mapping of tags to assign to the resource.

## Remarks
* Notice that all arguments are optional but you have to specify at least one argument.
* `domain_name_servers`, `netbios_name_servers`, `ntp_servers` are limited by AWS to maximum four servers only.
* To actually use the DHCP Options Set you need to associate it to a VPC using [`aws_vpc_dhcp_options_association`](/docs/providers/aws/r/vpc_dhcp_options_association.html).
* If you delete a DHCP Options Set, all VPCs using it will be associated to AWS's `default` DHCP Option Set.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the DHCP Options Set.

You can find more technical documentation about DHCP Options Set in the
official [AWS User Guide](http://docs.aws.amazon.com/AmazonVPC/latest/UserGuide/VPC_DHCP_Options.html).
