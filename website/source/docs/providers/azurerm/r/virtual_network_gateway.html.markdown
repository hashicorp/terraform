---
layout: "azurerm"
page_title: "Azure Resource Manager: azure_virtual_network_gateway"
sidebar_current: "docs-azurerm-resource-network-virtual-network-gateway"
description: |-
  Creates a new virtual network gateway to establish secure, cross-premises connectivity.
---

# azurerm\_virtual\_network\_gateway

Creates a new virtual network gateway to establish secure, cross-premises connectivity.

## Example Usage

```
resource "azurerm_resource_group" "test" {
    name = "test"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
  name = "test"
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  address_space = ["10.0.0.0/16"]
}

resource "azurerm_subnet" "test" {
  name = "GatewaySubnet"
  resource_group_name = "${azurerm_resource_group.test.name}"
  virtual_network_name = "${azurerm_virtual_network.test.name}"
  address_prefix = "10.0.1.0/24"
}

resource "azurerm_public_ip" "test" {
  name = "test"
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  public_ip_address_allocation = "Dynamic"
}

resource "azurerm_virtual_network_gateway" "test" {
  name = "test"
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"

  type = "Vpn"
  vpn_type = "RouteBased"

  active_active = false
  enable_bgp = false

	sku {
		name = "Basic"
		tier = "Basic"
	}

  ip_configuration {
    name = "vnetGatewayConfig"
    public_ip_address_id = "${azurerm_public_ip.test.id}"
    private_ip_address_allocation = "Dynamic"
    subnet_id = "${azurerm_subnet.test.id}"
  }

  vpn_client_configuration {
    address_space = [ "10.0.2.0/24" ]

    root_certificate {
      name = "DigiCert-Federated-ID-Root-CA"
      public_cert_data = <<EOF
MIIDuzCCAqOgAwIBAgIQCHTZWCM+IlfFIRXIvyKSrjANBgkqhkiG9w0BAQsFADBn
MQswCQYDVQQGEwJVUzEVMBMGA1UEChMMRGlnaUNlcnQgSW5jMRkwFwYDVQQLExB3
d3cuZGlnaWNlcnQuY29tMSYwJAYDVQQDEx1EaWdpQ2VydCBGZWRlcmF0ZWQgSUQg
Um9vdCBDQTAeFw0xMzAxMTUxMjAwMDBaFw0zMzAxMTUxMjAwMDBaMGcxCzAJBgNV
BAYTAlVTMRUwEwYDVQQKEwxEaWdpQ2VydCBJbmMxGTAXBgNVBAsTEHd3dy5kaWdp
Y2VydC5jb20xJjAkBgNVBAMTHURpZ2lDZXJ0IEZlZGVyYXRlZCBJRCBSb290IENB
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAvAEB4pcCqnNNOWE6Ur5j
QPUH+1y1F9KdHTRSza6k5iDlXq1kGS1qAkuKtw9JsiNRrjltmFnzMZRBbX8Tlfl8
zAhBmb6dDduDGED01kBsTkgywYPxXVTKec0WxYEEF0oMn4wSYNl0lt2eJAKHXjNf
GTwiibdP8CUR2ghSM2sUTI8Nt1Omfc4SMHhGhYD64uJMbX98THQ/4LMGuYegou+d
GTiahfHtjn7AboSEknwAMJHCh5RlYZZ6B1O4QbKJ+34Q0eKgnI3X6Vc9u0zf6DH8
Dk+4zQDYRRTqTnVO3VT8jzqDlCRuNtq6YvryOWN74/dq8LQhUnXHvFyrsdMaE1X2
DwIDAQABo2MwYTAPBgNVHRMBAf8EBTADAQH/MA4GA1UdDwEB/wQEAwIBhjAdBgNV
HQ4EFgQUGRdkFnbGt1EWjKwbUne+5OaZvRYwHwYDVR0jBBgwFoAUGRdkFnbGt1EW
jKwbUne+5OaZvRYwDQYJKoZIhvcNAQELBQADggEBAHcqsHkrjpESqfuVTRiptJfP
9JbdtWqRTmOf6uJi2c8YVqI6XlKXsD8C1dUUaaHKLUJzvKiazibVuBwMIT84AyqR
QELn3e0BtgEymEygMU569b01ZPxoFSnNXc7qDZBDef8WfqAV/sxkTi8L9BkmFYfL
uGLOhRJOFprPdoDIUBB+tmCl3oDcBy3vnUeOEioz8zAkprcb3GHwHAK+vHmmfgcn
WsfMLH4JCLa/tRYL+Rw/N3ybCkDp00s0WUZ+AoDywSl0Q/ZEnNY0MsFiw6LyIdbq
M/s/1JRtO3bDSzD9TazRVzn2oBqzSa8VgIo5C1nOnoAKJTlsClJKvIhnRlaLQqk=
EOF
    }

    revoked_certificate {
      name = "Verizon-Global-Root-CA"
      thumbprint = "912198EEF23DCAC40939312FEE97DD560BAE49B1"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the virtual network gateway. Changing the name
    forces a new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the virtual network gateway.

* `location` - (Required) The location/region where the virtual network gateway is
    created. Changing the location/region forces a new resource to be created.

* `type` - (Required) The type of the virtual network gateway. Valid options are
    `Vpn` or `ExpressRoute`. Changing the type forces a new resource to be created.

* `vpn_type` - (Optional) The routing type of the virtual network gateway. Valid
    options are `RouteBased` or `PolicyBased`. By default, a route based virtual
    network gateway will be created.

* `enable_bgp` - (Optional) If true, BGP (Border Gateway Protocol) will be enabled
    for this virtual network gateway. By default BGP is disabled.

* `active_active` - (Optional) If true, an active-active virtual network gateway
    will be created. An active-active gateway requires a `HighPerformance` or an
    `UltraPerformance` sku. By default, an active-standby gateway will be created.

* `default_local_network_gateway_id` -  (Optional) The ID of the local network gateway
    through which outbound Internet traffic from the virtual network in which the
    gateway is created will be routed (*forced tunneling*). Refer to the
    [Azure documentation on forced tunneling](https://docs.microsoft.com/en-us/azure/vpn-gateway/vpn-gateway-forced-tunneling-rm).
    By default, forced tunneling is not enabled.

* `sku` - (Required) Configuration of the size and capacity of the virtual network
    gateway. The `sku` block supports fields documented below.

* `ip_configuration` (Required) One or two `ip_configuration` blocks documented below.
    An active-standby gateway requires exactly one `ip_configuration` block whereas
    an active-active gateway requires exactly two `ip_configuration` blocks.

* `vpn_client_configuration` (Optional) A `vpn_client_configuration` block which
    is documented below. In this block the virtual network gateway can be configured
    to accept IPSec point-to-site connections.

* `tags` - (Optional) A mapping of tags to assign to the resource.

The `sku` block supports:

* `name` - (Required) The sku name of the virtual network gateway instance. Valid
    options are `Basic`, `Standard`, `HighPerformance`, and `UltraPerformance`.

* `tier` - (Required) The sku tier of the virtual network gateway instance. Valid
    options are `Basic`, `Standard`, `HighPerformance`, and `UltraPerformance`.

* `capacity` - (Optional) The capacity of the virtual network gateway. If not
    specified, the maximum capacity of the chosen sku will be assumed.

The `ip_configuration` block supports:

* `name` - (Optional) A user-defined name of the IP.

* `private_ip_address_allocation` - (Optional) Defines how the private IP address
    of the gateways virtual interface is assigned. Valid options are `Static` or
    `Dynamic`. By default dynamic allocation will be used.

* `subnet_id` - (Required) The ID of the gateway subnet of a virtual network in
    which the virtual network gateway will be created. It is mandatory that
    the associated subnet is named `GatewaySubnet`. Therefore, each virtual
    network can contain at most a single virtual network gateway.

* `public_ip_address_id` - (Optional) The ID of the public ip address to associate
    with the virtual network gateway.

The `vpn_client_configuration` block supports:

* `address_space` - (Required) The address space out of which ip addresses for
    vpn clients will be taken. You can provide more than one address space, e.g.
    in CIDR notation.

* `root_certificate` - (Required) One or more `root_certificate` blocks which are
    defined below. These root certificates are used to sign the client certificate
    used by the VPN clients to connect to the gateway.

* `revoked_certificate` - (Optional) One or more `revoked_certificate` blocks which
    are defined below.

The `bgp_settings` block supports:

* `asn` - (Optional) The Autonomous System Number (ASN) to use as part of the BGP.

* `peering_address` - (Optional) The BGP peer IP address of the virtual network
    gateway. This address is needed to configure the created gateway as a BGP Peer
    on the on-premises VPN devices. The IP address must be part of the subnet of
    the virtual network gateway.

* `peer_weight` - (Optional) The weight added to routes which have been learned
    through BGP peering. Valid values can be between 0 and 100.

The `root_certificate` block supports:

* `name` - A user-defined name of the root certificate.

* `public_cert_data` - (Required) The public certificate of the root certificate
    authority. The certificate must be provided in Base-64 encoded X.509 format
    (PEM). In particular, this argument *must not* include the
    `-----BEGIN CERTIFICATE-----` or `-----END CERTIFICATE-----` markers.

The `root_revoked_certificate` block supports:

* `name` - A user-defined name of the revoked certificate.

* `public_cert_data` - (Required) The SHA1 thumbprint of the certificate to be
    revoked.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the virtual network gateway.

* `name` - The name of the virtual network gateway.

* `resource_group_name` - The name of the resource group in which to create the virtual network gateway.

* `location` - The location/region where the virtual network gateway is created


## Import

Virtual Network Gateways can be imported using the `resource id`, e.g.

```
terraform import azurerm_virtual_network_gateway.testGateway /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myGroup1/providers/Microsoft.Network/virtualNetworkGateways/myGateway1
```
