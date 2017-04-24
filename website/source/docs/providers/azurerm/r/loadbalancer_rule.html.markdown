---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_lb_rule"
sidebar_current: "docs-azurerm-resource-loadbalancer-rule"
description: |-
  Create a LoadBalancer Rule.
---

# azurerm\_lb\_rule

Create a LoadBalancer Rule.

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

resource "azurerm_lb_rule" "test" {
  resource_group_name            = "${azurerm_resource_group.test.name}"
  loadbalancer_id                = "${azurerm_lb.test.id}"
  name                           = "LBRule"
  protocol                       = "Tcp"
  frontend_port                  = 3389
  backend_port                   = 3389
  frontend_ip_configuration_name = "PublicIPAddress"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the LB Rule.
* `resource_group_name` - (Required) The name of the resource group in which to create the resource.
* `loadbalancer_id` - (Required) The ID of the LoadBalancer in which to create the Rule.
* `frontend_ip_configuration_name` - (Required) The name of the frontend IP configuration to which the rule is associated.
* `protocol` - (Required) The transport protocol for the external endpoint. Possible values are `Udp` or `Tcp`.
* `frontend_port` - (Required) The port for the external endpoint. Port numbers for each Rule must be unique within the Load Balancer. Possible values range between 1 and 65534, inclusive.
* `backend_port` - (Required) The port used for internal connections on the endpoint. Possible values range between 1 and 65535, inclusive.
* `backend_address_pool_id` - (Optional) A reference to a Backend Address Pool over which this Load Balancing Rule operates.
* `probe_id` - (Optional) A reference to a Probe used by this Load Balancing Rule.
* `enable_floating_ip` - (Optional) Floating IP is pertinent to failover scenarios: a "floating” IP is reassigned to a secondary server in case the primary server fails. Floating IP is required for SQL AlwaysOn.
* `idle_timeout_in_minutes` - (Optional) Specifies the timeout for the Tcp idle connection. The value can be set between 4 and 30 minutes. The default value is 4 minutes. This element is only used when the protocol is set to Tcp.
* `load_distribution` - (Optional) Specifies the load balancing distribution type to be used by the Load Balancer. Possible values are: Default – The load balancer is configured to use a 5 tuple hash to map traffic to available servers. SourceIP – The load balancer is configured to use a 2 tuple hash to map traffic to available servers. SourceIPProtocol – The load balancer is configured to use a 3 tuple hash to map traffic to available servers.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the LoadBalancer to which the resource is attached.

## Import

Load Balancer Rules can be imported using the `resource id`, e.g.

```
terraform import azurerm_lb_rule.test /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/group1/providers/Microsoft.Network/loadBalancers/lb1/loadBalancingRules/rule1
```
