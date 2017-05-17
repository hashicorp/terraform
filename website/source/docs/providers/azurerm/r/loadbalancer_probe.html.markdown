---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_lb_probe"
sidebar_current: "docs-azurerm-resource-loadbalancer-probe"
description: |-
  Create a LoadBalancer Probe Resource.
---

# azurerm\_lb\_probe

Create a LoadBalancer Probe Resource.

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

resource "azurerm_lb_probe" "test" {
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id     = "${azurerm_lb.test.id}"
  name                = "ssh-running-probe"
  port                = 22
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the Probe.
* `resource_group_name` - (Required) The name of the resource group in which to create the resource.
* `loadbalancer_id` - (Required) The ID of the LoadBalancer in which to create the NAT Rule.
* `protocol` - (Optional) Specifies the protocol of the end point. Possible values are `Http` or `Tcp`. If Tcp is specified, a received ACK is required for the probe to be successful. If Http is specified, a 200 OK response from the specified URI is required for the probe to be successful.
* `port` - (Required) Port on which the Probe queries the backend endpoint. Possible values range from 1 to 65535, inclusive.
* `request_path` - (Optional) The URI used for requesting health status from the backend endpoint. Required if protocol is set to Http. Otherwise, it is not allowed.
* `interval_in_seconds` - (Optional) The interval, in seconds between probes to the backend endpoint for health status. The default value is 15, the minimum value is 5.
* `number_of_probes` - (Optional) The number of failed probe attempts after which the backend endpoint is removed from rotation. The default value is 2. NumberOfProbes multiplied by intervalInSeconds value must be greater or equal to 10.Endpoints are returned to rotation when at least one probe is successful.


## Attributes Reference

The following attributes are exported:

* `id` - The ID of the LoadBalancer to which the resource is attached.

## Import

Load Balancer Probes can be imported using the `resource id`, e.g.

```
terraform import azurerm_lb_probe.test /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/group1/providers/Microsoft.Network/loadBalancers/lb1/probes/probe1
```
