---
layout: "alicloud"
page_title: "Alicloud: alicloud_nat_gateway"
sidebar_current: "docs-alicloud-resource-nat-gateway"
description: |-
  Provides a resource to create a VPC NAT Gateway.
---

# alicloud\_nat\_gateway

Provides a resource to create a VPC NAT Gateway.

~> **NOTE:** alicloud_nat_gateway must depends on alicloud_vswitch.


## Example Usage

Basic usage

```
resource "alicloud_vpc" "vpc" {
  name       = "tf_test_foo"
  cidr_block = "172.16.0.0/12"
}

resource "alicloud_vswitch" "vsw" {
  vpc_id            = "${alicloud_vpc.vpc.id}"
  cidr_block        = "172.16.0.0/21"
  availability_zone = "cn-beijing-b"
}

resource "alicloud_nat_gateway" "nat_gateway" {
  vpc_id = "${alicloud_vpc.vpc.id}"
  spec   = "Small"
  name   = "test_foo"

  bandwidth_packages = [{
    ip_count  = 1
    bandwidth = 5
    zone      = "cn-beijing-b"
  },
    {
      ip_count  = 2
      bandwidth = 10
      zone      = "cn-beijing-b"
    },
  ]

  depends_on = [
    "alicloud_vswitch.vsw",
  ]
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required, Forces New Resorce) The VPC ID.
* `spec` - (Required, Forces New Resorce) The specification of the nat gateway. Valid values are `Small`, `Middle` and `Large`. Details refer to [Nat Gateway Specification](https://help.aliyun.com/document_detail/42757.html?spm=5176.doc32322.6.559.kFNBzv)
* `name` - (Optional) Name of the nat gateway. The value can have a string of 2 to 128 characters, must contain only alphanumeric characters or hyphens, such as "-",".","_", and must not begin or end with a hyphen, and must not begin with http:// or https://. Defaults to null.
* `description` - (Optional) Description of the nat gateway, This description can have a string of 2 to 256 characters, It cannot begin with http:// or https://. Defaults to null.
* `bandwidth_packages` - (Required) A list of bandwidth packages for the nat gatway.

## Block bandwidth package

The bandwidth package mapping supports the following:

* `ip_count` - (Required) The IP number of the current bandwidth package. Its value range from 1 to 50.
* `bandwidth` - (Required) The bandwidth value of the current bandwidth package. Its value range from 5 to 5000.
* `zone` - (Optional) The AZ for the current bandwidth. If this value is not specified, Terraform will set a random AZ.
* `public_ip_addresses` - (Computer) The public ip for bandwidth package. the public ip count equal `ip_count`, multi ip would complex with ",", such as "10.0.0.1,10.0.0.2".

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the nat gateway.
* `name` - The name of the nat gateway.
* `description` - The description of the nat gateway.
* `spec` - The specification of the nat gateway.
* `vpc_id` - The VPC ID for the nat gateway.
* `bandwidth_package_ids` - A list ID of the bandwidth packages, and split them with commas
* `snat_table_ids` - The nat gateway will auto create a snap and forward item, the `snat_table_ids` is the created one.
* `forward_table_ids` - The nat gateway will auto create a snap and forward item, the `forward_table_ids` is the created one.
