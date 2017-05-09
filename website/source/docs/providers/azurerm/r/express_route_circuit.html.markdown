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
resource "azurerm_resource_group" "test" {
  name     = "exprtTest"
  location = "West US"
}

resource "azurerm_express_route_circuit" "test" {
  name                     = "expressRoute1"
  resource_group_name      = "${azurerm_resource_group.test.name}"
  location                 = "West US"
  service_provider_name    = "Equinix"
  peering_location         = "Silicon Valley"
  bandwidth_in_mbps        = 50
  sku {
    tier   = "Standard"
    family = "MeteredData"
  }
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

* `bandwidth_in_mbps` - (Required) The bandwidth in Mbps of the circuit being created. Once you increase your bandwidth, 
    you will not be able to decrease it to its previous value.

* `sku` - (Required) Chosen SKU of ExpressRoute circuit as documented below.

* `allow_classic_operations` - (Optional) Allow the circuit to interact with classic (RDFE) resources.
    The default value is false.

* `tags` - (Optional) A mapping of tags to assign to the resource.

`sku` supports the following:

* `tier` - (Required) The service tier. Value must be either "Premium" or "Standard".

* `family` - (Required) The billing mode. Value must be either "MeteredData" or "UnlimitedData". 
   Once you set the billing model to "UnlimitedData", you will not be able to switch to "MeteredData".

## Attributes Reference

The following attributes are exported:

* `id` - The Resource ID of the ExpressRoute circuit.
* `service_provider_provisioning_state` - The ExpressRoute circuit provisioning state from your chosen service provider. 
    Possible values are "NotProvisioned", "Provisioning", "Provisioned", and "Deprovisioning".
* `service_key` - The string needed by the service provider to provision the ExpressRoute circuit.

## Import

ExpressRoute circuits can be imported using the `resource id`, e.g.

```
terraform import azurerm_express_route_circuit.myExpressRoute /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mygroup1/providers/Microsoft.Network/expressRouteCircuits/myExpressRoute
```
