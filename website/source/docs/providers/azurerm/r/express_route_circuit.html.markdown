---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_express_route_circuit"
sidebar_current: "docs-azurerm-resource-express-route-circuit"
description: |-
  Creates an ExpressRoute circuit.
---

# azurerm\_express\_route\_circuit

Creates an ExpressRoute circuit.

## Example Usage

```hcl
resource "azurerm_express_route_circuit" "test" {
  name                     = "expressRoute1"
  resource_group_name      = "${azurerm_resource_group.test.name}"
  location                 = "West US"
  service_provider_name    = "Equinix"
  peering_location         = "Silicon Valley"
  bandwidth_in_mbps        = 50
  sku_tier                 = "Standard"
  sku_family               = "MeteredData"
  allow_classic_operations = false

  tags {
    environment = "Production"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the ExpressRoute circuit. Changing this forces a
    new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the namespace. Changing this forces a new resource to be created.

* `location` - (Required) Specifies the supported Azure location where the resource exists.
    Changing this forces a new resource to be created.

* `service_provider_name` - (Required) The name of the ExpressRoute Service Provider.

* `peering_location` - (Required) The name of the peering location and not the ARM resource location.

* `bandwidth_in_mbps` - (Required) The bandwidth in Mbps of the circuit being created.

* `sku_tier` - (Optional) Chosen SKU Tier of ExpressRoute circuit. Value must be either "Premium" or "Standard". 
    The default value is "Standard".

* `sku_family` - (Optional) Chosen SKU family of ExpressRoute circuit. 
    Value must be either "MeteredData" or "UnlimitedData". The default value is "MeteredData".

* `allow_classic_operations` - (Optional) Allow the circuit to interact with classic (RDFE) resources.
    The default value is false.

* `tags` - (Optional) A mapping of tags to assign to the resource.

## Attributes Reference

The following attributes are exported:

* `id` - The Resource ID of the ExpressRoute circuit.
* `service_provider_provisioning_state` - The ServiceProviderProvisioningState state of the resource. 
    Possible values are "NotProvisioned", "Provisioning", "Provisioned", and "Deprovisioning".
* `service_key` - The string needed by the service provider to provision the ExpressRoute circuit.

## Import

ExpressRoute circuits can be imported using the `resource id`, e.g.

```
terraform import azurerm_express_route_circuit.myExpressRoute /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mygroup1/providers/Microsoft.Network/expressRouteCircuits/myExpressRoute
```
