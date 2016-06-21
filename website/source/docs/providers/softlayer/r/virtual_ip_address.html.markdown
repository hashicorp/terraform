---
layout: "softlayer"
page_title: "SoftLayer: softlayer_loadbalancer_virtual_ip_address"
sidebar_current: "docs-softlayer-resource-load-balancer-virtual-ip-address"
description: |-
  Provides Softlayer's LoadBalancer Virtual IP Address
---

# softlayer_loadbalancer_virtual_ip_address

Create, update, and delete Softlayer Load Balancer Virtual IP Addresses. For additional details please refer to the [API documentation](http://sldn.softlayer.com/reference/datatypes/SoftLayer_Network_LoadBalancer_VirtualIpAddress).
## Example Usage

```
resource "softlayer_network_loadbalancer_virtualipaddress" "testacc_vip" {
    name = "test_load_balancer_vip"
    nad_controller_id = "${softlayer_network_application_delivery_controller.testacc_foobar_nadc.id}"
    load_balancing_method = "lc"
    source_port = 80
    virtual_ip_address = "${softlayer_virtual_guest.terraform-acceptance-test-1.ipv4_address}"
    type = "HTTP"
}
```

## Argument Reference

* `name` | *string*
    * (Required) The unique identifier for the Network Load Balancer Virtual IP Address.
* `nad_controller_id` | *int*
    * (Required) The ID of the Network Application Delivery Controller that the Load Balancer Virtual IP Address will be assigned to.
* `load_balancing_method` | *string*
    * (Required) The Network Application Delivery Controller load balancing method. See [the documentation](http://sldn.softlayer.com/reference/datatypes/SoftLayer_Network_LoadBalancer_VirtualIpAddress) for details.
* `virtual_ip_address` | *string*
    * (Required) The public facing IP address for the Load Balancer Virtual IP.
* `source_port` | *int*
    * (Required) The source port for the Load Balancer Virtual IP Address.
* `type` | *string*
    * (Required) The connection type for the Load Balancer Virtual IP Address. Accepted values are `HTTP`, `FTP`, `TCP`, `UDP`, and `DNS`.
* `security_certificate_id` | *int*
    * (Optional) The id of the Security Certificate to be used when SSL is enabled.

## Attributes Reference

* `id` - The Load Balancer Virtual IPs unique identifier.
* `connection_limit` - The sum of the connection limit values of the Load Balancer Services associated with this Load Balancer Virtual IP Address.
* `modify_date` - The most recent time that the Load Balancer Virtual IP Address was modified.
