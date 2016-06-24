---
layout: "softlayer"
page_title: "SoftLayer: softlayer_lb_vpx"
sidebar_current: "docs-softlayer-resource-lb-vpx"
description: |-
  Provides Softlayer's Load Balancer VPX
---

# softlayer_lb_vpx

Create, update, and destroy SoftLayer VPX Load Balancers.

_Please Note_: SoftLayer VPX Load Balancer consists of Citrix Netscaler VPX devices (virtual), these are currently priced on a per-month basis, so please use caution when creating the resource as the cost for an entire month is incurred immediately upon creation. For more information on pricing please see this [link](http://www.softlayer.com/network-appliances), under the Citrix log, click "see more pricing" for a current price matrix.

You can also use this REST URL to get a listing of VPX choices along with version numbers, speed and plan type:

```
https://{{userName}}:{{apiKey}}@api.softlayer.com/rest/v3/SoftLayer_Product_Package/192/getItems.json?objectMask=id;capacity;description;units;keyName;prices.id;prices.categories.id;prices.categories.name
```


## Example Usage | [SLDN](http://sldn.softlayer.com/reference/datatypes/SoftLayer_Network_Application_Delivery_Controller)

```
resource "softlayer_lb_vpx" "test_vpx" {
    datacenter = "DALLAS06"
    speed = 10
    version = "10.1"
    plan = "Standard"
    ip_count = 2
}
```

## Argument Reference

* `datacenter` | *string*
    * (Required) Specifies which datacenter the VPX Load Balancer is to be provisioned in. Accepted values can be found [here](http://www.softlayer.com/data-centers).
* `speed` | *int*
    * (Required) The speed in Mbps. Accepted values are `10`, `200`, and `1000`.
* `version` | *string*
    * (Required) The VPX Load Balancer version. Accepted values are `10.1` and `10.5`.
* `plan` | *string*
    * (Required) The VPX Load Balancer plan. Accepted values are `Standard` and `Platinum`.
* `ip_count` | *int*
    * (Required) The number of static public IP addresses assigned to the VPX Load Balancer. Accepted values are `2`, `4`, `8`, and `16`.

## Attributes Reference

* `id` - A VPX Load Balancer's internal identifier.
* `name` - A VPX Load Balancer's internal name.
