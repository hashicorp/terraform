---
layout: "azure"
page_title: "Azure: azure_virtual_network"
sidebar_current: "docs-azure-resource-virtual-network"
description: |-
  Creates a new virtual network including any configured subnets. Each subnet can optionally be configured with a security group to be associated with the subnet.
---

# azure\_virtual\_network

Creates a new virtual network including any configured subnets. Each subnet can
optionally be configured with a security group to be associated with the subnet.

## Example Usage

```hcl
resource "azure_virtual_network" "default" {
  name          = "test-network"
  address_space = ["10.1.2.0/24"]
  location      = "West US"

  subnet {
    name           = "subnet1"
    address_prefix = "10.1.2.0/25"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the virtual network. Changing this forces a
    new resource to be created.

* `address_space` - (Required) The address space that is used the virtual
    network. You can supply more than one address space. Changing this forces
    a new resource to be created.

* `location` - (Required) The location/region where the virtual network is
    created. Changing this forces a new resource to be created.

* `dns_servers` - (Optional) List of names of DNS servers previously registered
    on Azure.

* `subnet` - (Required) Can be specified multiple times to define multiple
    subnets. Each `subnet` block supports fields documented below.

The `subnet` block supports:

* `name` - (Required) The name of the subnet.

* `address_prefix` - (Required) The address prefix to use for the subnet.

* `security_group` - (Optional) The Network Security Group to associate with
    the subnet.

## Attributes Reference

The following attributes are exported:

* `id` - The virtual NetworkConfiguration ID.
