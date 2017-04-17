---
layout: "azurerm"
page_title: "Azure Resource Manager: azure_virtual_network"
sidebar_current: "docs-azurerm-resource-network-virtual-network"
description: |-
  Creates a new virtual network including any configured subnets. Each subnet can optionally be configured with a security group to be associated with the subnet.
---

# azurerm\_virtual\_network

Creates a new virtual network including any configured subnets. Each subnet can
optionally be configured with a security group to be associated with the subnet.

## Example Usage

```hcl
resource "azurerm_virtual_network" "test" {
  name                = "virtualNetwork1"
  resource_group_name = "${azurerm_resource_group.test.name}"
  address_space       = ["10.0.0.0/16"]
  location            = "West US"
  dns_servers         = ["10.0.0.4", "10.0.0.5"]

  subnet {
    name           = "subnet1"
    address_prefix = "10.0.1.0/24"
  }

  subnet {
    name           = "subnet2"
    address_prefix = "10.0.2.0/24"
  }

  subnet {
    name           = "subnet3"
    address_prefix = "10.0.3.0/24"
    security_group = "${azurerm_network_security_group.test.id}"
  }

  tags {
    environment = "Production"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the virtual network. Changing this forces a
    new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the virtual network.

* `address_space` - (Required) The address space that is used the virtual
    network. You can supply more than one address space. Changing this forces
    a new resource to be created.

* `location` - (Required) The location/region where the virtual network is
    created. Changing this forces a new resource to be created.

* `dns_servers` - (Optional) List of IP addresses of DNS servers

* `subnet` - (Optional) Can be specified multiple times to define multiple
    subnets. Each `subnet` block supports fields documented below.

* `tags` - (Optional) A mapping of tags to assign to the resource.

The `subnet` block supports:

* `name` - (Required) The name of the subnet.

* `address_prefix` - (Required) The address prefix to use for the subnet.

* `security_group` - (Optional) The Network Security Group to associate with
    the subnet. (Referenced by `id`, ie. `azurerm_network_security_group.test.id`)

## Attributes Reference

The following attributes are exported:

* `id` - The virtual NetworkConfiguration ID.

* `name` - The name of the virtual network.

* `resource_group_name` - The name of the resource group in which to create the virtual network.

* `location` - The location/region where the virtual network is created

* `address_space` - The address space that is used the virtual network.


## Import

Virtual Networks can be imported using the `resource id`, e.g.

```
terraform import azurerm_virtual_network.testNetwork /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mygroup1/providers/Microsoft.Network/virtualNetworks/myvnet1
```
