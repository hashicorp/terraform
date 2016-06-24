---
layout: "softlayer"
page_title: "SoftLayer: softlayer_loadbalancer_service"
sidebar_current: "docs-softlayer-resource-lb-vpx-service"
description: |-
  Provides Softlayer's Load Balancer VPX Service
---

# softlayer_lb_vpx_service

Create, update, and delete Softlayer VPX Load Balancer Services. For additional details please refer to the [API documentation](http://sldn.softlayer.com/reference/datatypes/SoftLayer_Network_LoadBalancer_Service).
## Example Usage

```
resource "softlayer_lb_vpx_service" "test_service" {
  name = "test_load_balancer_service"
  vip_id = "${softlayer_lb_vpx_vip.testacc_vip.id}"
  destination_ip_address = "${softlayer_virtual_guest.terraform-acceptance-test-2.ipv4_address}"
  destination_port = 89
  weight = 55
  connection_limit = 5000
  health_check = "HTTP"
}
```

## Argument Reference

* `name` | *string*
    * (Required) The unique identifier for the VPX Load Balancer Service.
* `vip_id` | *string*
    * (Required) The ID of the VPX Load Balancer Virtual IP Address that the VPX Load Balancer Service is assigned to.
* `destination_ip_address` | *string*
    * (Required) The IP address of the server traffic will be directed to.
* `destination_port` | *int*
    * (Required) The destination port of the server traffic will be directed to.
* `weight` | *int*
    * (Required) Set the weight of this VPX Load Balancer service. Affects the choices the VPX Load Balancer makes between the various services. See [the documentation](http://sldn.softlayer.com/reference/datatypes/SoftLayer_Network_LoadBalancer_Service) for details.
* `connection_limit` | *int*
    * (Required) Set the connection limit for this service.
* `health_check` | *string*
    * (Required) Set the health check for the VPX Load Balancer Service. See [the documentation](http://sldn.softlayer.com/reference/datatypes/SoftLayer_Network_LoadBalancer_Service) for details.

## Attributes Reference

* `id` - The VPX Load Balancer Service unique identifier.
