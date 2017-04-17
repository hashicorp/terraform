---
layout: "azure"
page_title: "Azure: azure_local_network_connection"
sidebar_current: "docs-azure-resource-local-network-connection"
description: |-
  Defines a new connection to a remote network through a VPN tunnel.
---

# azure\_local\_network\_connection

Defines a new connection to a remote network through a VPN tunnel.

## Example Usage

```hcl
resource "azure_local_network_connection" "localnet" {
  name                   = "terraform-local-network-connection"
  vpn_gateway_address    = "45.12.189.2"
  address_space_prefixes = ["10.10.10.0/24", "10.10.11.0/24"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name by which this local network connection will
    be referenced by. Changing this forces a new resource to be created.

* `vpn_gateway_address` - (Required) The public IPv4 of the VPN endpoint.

* `address_space_prefixes` - (Required) List of address spaces accessible
    through the VPN connection. The elements are in the CIDR format.

## Attributes Reference

The following attributes are exported:

* `id` - The local network connection ID.
