---
layout: "aws"
page_title: "AWS: aws_vpc"
sidebar_current: "docs-aws-resource-vpc"
description: |-
  Provides an VPC resource.
---

# aws\_vpc

Provides an VPC resource.

## Example Usage

Basic usage:

```
resource "aws_vpc" "main" {
    cidr_block = "10.0.0.0/16"
}
```

Basic usage with tags:

```
resource "aws_vpc" "main" {
	cidr_block = "10.0.0.0/16"
	instance_tenancy = "dedicated"

	tags {
		Name = "main"
	}
}
```

## Argument Reference

The following arguments are supported:

* `cidr_block` - (Required) The CIDR block for the VPC.
* `instance_tenancy` - (Optional) A tenancy option for instances launched into the VPC
* `enable_dns_support` - (Optional) A boolean flag to enable/disable DNS support in the VPC. Defaults true.
* `enable_dns_hostnames` - (Optional) A boolean flag to enable/disable DNS hostnames in the VPC. Defaults false.
* `enable_classiclink` - (Optional) A boolean flag to enable/disable ClassicLink 
  for the VPC. Only valid in regions and accounts that support EC2 Classic.
  See the [ClassicLink documentation][1] for more information. Defaults false.
* `tags` - (Optional) A mapping of tags to assign to the resource.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the VPC
* `cidr_block` - The CIDR block of the VPC
* `instance_tenancy` - Tenancy of instances spin up within VPC.
* `enable_dns_support` - Whether or not the VPC has DNS support
* `enable_dns_hostnames` - Whether or not the VPC has DNS hostname support
* `enable_classiclink` - Whether or not the VPC has Classiclink enabled
* `main_route_table_id` - The ID of the main route table associated with
     this VPC. Note that you can change a VPC's main route table by using an
     [`aws_main_route_table_association`](/docs/providers/aws/r/main_route_table_assoc.html).
* `default_network_acl_id` - The ID of the network ACL created by default on VPC creation
* `default_security_group_id` - The ID of the security group created by default on VPC creation


[1]: http://docs.aws.amazon.com/fr_fr/AWSEC2/latest/UserGuide/vpc-classiclink.html
