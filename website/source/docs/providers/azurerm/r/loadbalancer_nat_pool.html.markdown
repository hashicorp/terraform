---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_lb_nat_pool"
sidebar_current: "docs-azurerm-resource-loadbalancer-nat-pool"
description: |-
  Create a LoadBalancer NAT Pool.
---

# azurerm\_lb\_nat\_pool

Create a LoadBalancer NAT pool.

~> **NOTE When using this resource, the LoadBalancer needs to have a FrontEnd IP Configuration Attached

## Example Usage

```hcl
resource "azurerm_resource_group" "test" {
  name     = "LoadBalancerRG"
  location = "West US"
}

resource "azurerm_public_ip" "test" {
  name                         = "PublicIPForLB"
  location                     = "West US"
  resource_group_name          = "${azurerm_resource_group.test.name}"
  public_ip_address_allocation = "static"
}

resource "azurerm_lb" "test" {
  name                = "TestLoadBalancer"
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"

  frontend_ip_configuration {
    name                 = "PublicIPAddress"
    public_ip_address_id = "${azurerm_public_ip.test.id}"
  }
}

resource "azurerm_lb_nat_pool" "test" {
  resource_group_name            = "${azurerm_resource_group.test.name}"
  loadbalancer_id                = "${azurerm_lb.test.id}"
  name                           = "SampleApplication Pool"
  protocol                       = "Tcp"
  frontend_port_start            = 80
  frontend_port_end              = 81
  backend_port                   = 8080
  frontend_ip_configuration_name = "PublicIPAddress"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the NAT pool.
* `resource_group_name` - (Required) The name of the resource group in which to create the resource.
* `loadbalancer_id` - (Required) The ID of the LoadBalancer in which to create the NAT pool.
* `frontend_ip_configuration_name` - (Required) The name of the frontend IP configuration exposing this rule.
* `protocol` - (Required) The transport protocol for the external endpoint. Possible values are `Udp` or `Tcp`.
* `frontend_port_start` - (Required) The first port number in the range of external ports that will be used to provide Inbound Nat to NICs associated with this Load Balancer. Possible values range between 1 and 65534, inclusive.
* `frontend_port_end` - (Required) The last port number in the range of external ports that will be used to provide Inbound Nat to NICs associated with this Load Balancer. Possible values range between 1 and 65534, inclusive.
* `backend_port` - (Required) The port used for the internal endpoint. Possible values range between 1 and 65535, inclusive.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the LoadBalancer to which the resource is attached.

## Import

Load Balancer NAT Pools can be imported using the `resource id`, e.g.

```
terraform import azurerm_lb_nat_pool.test /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/group1/providers/Microsoft.Network/loadBalancers/lb1/inboundNatPools/pool1
```
