---
layout: "aws"
page_title: "AWS: aws_default_vpc"
sidebar_current: "docs-aws-resource-default-vpc"
description: |-
  Manage the default VPC resource.
---

# aws\_default\_vpc

Provides a resource to manage the [default AWS VPC](http://docs.aws.amazon.com/AmazonVPC/latest/UserGuide/default-vpc.html)
in the current region.

For AWS accounts created after 2013-12-04, each region comes with a Default VPC.
**This is an advanced resource**, and has special caveats to be aware of when
using it. Please read this document in its entirety before using this resource.

The `aws_default_vpc` behaves differently from normal resources, in that
Terraform does not _create_ this resource, but instead "adopts" it
into management. 

## Example Usage

Basic usage with tags:

```
resource "aws_default_vpc" "default" {
	tags {
		Name = "Default VPC"
	}
}
```

## Argument Reference

The arguments of an `aws_default_vpc` differ slightly from `aws_vpc` 
resources. Namely, the `cidr_block`, `instance_tenancy` and `assign_generated_ipv6_cidr_block`
arguments are computed. The following arguments are still supported: 

* `enable_dns_support` - (Optional) A boolean flag to enable/disable DNS support in the VPC. Defaults true.
* `enable_dns_hostnames` - (Optional) A boolean flag to enable/disable DNS hostnames in the VPC. Defaults false.
* `enable_classiclink` - (Optional) A boolean flag to enable/disable ClassicLink 
  for the VPC. Only valid in regions and accounts that support EC2 Classic.
  See the [ClassicLink documentation][1] for more information. Defaults false.
* `tags` - (Optional) A mapping of tags to assign to the resource.

### Removing `aws_default_vpc` from your configuration

The `aws_default_vpc` resource allows you to manage a region's default VPC,
but Terraform cannot destroy it. Removing this resource from your configuration
will remove it from your statefile and management, but will not destroy the VPC.
You can resume managing the VPC via the AWS Console.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the VPC
* `cidr_block` - The CIDR block of the VPC
* `instance_tenancy` - Tenancy of instances spin up within VPC.
* `enable_dns_support` - Whether or not the VPC has DNS support
* `enable_dns_hostnames` - Whether or not the VPC has DNS hostname support
* `enable_classiclink` - Whether or not the VPC has Classiclink enabled
* `assign_generated_ipv6_cidr_block` - Whether or not an Amazon-provided IPv6 CIDR 
block with a /56 prefix length for the VPC was assigned
* `main_route_table_id` - The ID of the main route table associated with
     this VPC. Note that you can change a VPC's main route table by using an
     [`aws_main_route_table_association`](/docs/providers/aws/r/main_route_table_assoc.html)
* `default_network_acl_id` - The ID of the network ACL created by default on VPC creation
* `default_security_group_id` - The ID of the security group created by default on VPC creation
* `default_route_table_id` - The ID of the route table created by default on VPC creation
* `ipv6_association_id` - The association ID for the IPv6 CIDR block of the VPC
* `ipv6_cidr_block` - The IPv6 CIDR block of the VPC


[1]: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/vpc-classiclink.html
