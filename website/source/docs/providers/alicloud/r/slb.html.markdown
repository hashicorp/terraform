---
layout: "alicloud"
page_title: "Alicloud: alicloud_slb"
sidebar_current: "docs-alicloud-resource-slb"
description: |-
  Provides an Application Load Banlancer resource.
---

# alicloud\_slb

Provides an Application Load Balancer resource.

## Example Usage

```
# Create a new load balancer for classic
resource "alicloud_slb" "classic" {
	name = "test-slb-tf"
	internet = true
	internet_charge_type = "paybybandwidth"
	bandwidth = 5
	listener = [
	{
		"instance_port" = "2111"
		"lb_port" = "21"
		"lb_protocol" = "tcp"
		"bandwidth" = "5"
	},{
		"instance_port" = "8000"
		"lb_port" = "80"
		"lb_protocol" = "http"
		"bandwidth" = "5"
	},{
		"instance_port" = "1611"
		"lb_port" = "161"
		"lb_protocol" = "udp"
		"bandwidth" = "5"
	}]
}

# Create a new load balancer for VPC
resource "alicloud_vpc" "default" {
	# Other parameters...
}

resource "alicloud_vswitch" "default" {
	# Other parameters...
}

resource "alicloud_slb" "vpc" {
	name = "test-slb-tf"
	vswitch_id = "${alicloud_vswitch.default.id}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) The name of the SLB. This name must be unique within your AliCloud account, can have a maximum of 80 characters, 
must contain only alphanumeric characters or hyphens, such as "-","/",".","_", and must not begin or end with a hyphen. If not specified, 
Terraform will autogenerate a name beginning with `tf-lb`.
* `internet` - (Optional, Forces New Resource) If true, the SLB addressType will be internet, false will be intranet, Default is false. If load balancer launched in VPC, this value must be "false".
* `internet_charge_type` - (Optional, Forces New Resource) Valid
  values are `paybybandwidth`, `paybytraffic`. If this value is "paybybandwidth", then argument "internet" must be "true". Default is "paybytraffic". If load balancer launched in VPC, this value must be "paybytraffic".
* `bandwidth` - (Optional) Valid
  value is between 1 and 1000, If argument "internet_charge_type" is "paybytraffic", then this value will be ignore.
* `listener` - (Optional) Additional SLB listener. See [Block listener](#block-listener) below for details.
* `vswitch_id` - (Required for a VPC SLB, Forces New Resource) The VSwitch ID to launch in.

## Block listener

The listener mapping supports the following:

* `instance_port` - (Required) The port on which the backend servers are listening. Valid value is between 1 to 65535.
* `lb_port` - (Required) The port on which the load balancer is listening. Valid value is between 1 to 65535.
* `lb_protocol` - (Required) The protocol to listen on. Valid values are `http` and and `tcp` and `udp`. 
* `bandwidth` - (Required) The bandwidth on which the load balancer is  listening. Valid values is -1 or between 1 and 1000. If -1, the bindwidth will haven’t upper limit.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the load balancer.
* `name` - The name of the load balancer.
* `internet` - The internet of the load balancer.
* `internet_charge_type` - The internet_charge_type of the load balancer.
* `bandwidth` - The bandwidth of the load balancer.
* `vswitch_id` - The VSwitch ID of the load balancer. Only available on SLB launched in a VPC.
* `address` - The IP address of the load balancer.