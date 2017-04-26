---
layout: "opc"
page_title: "Oracle: opc_compute_ip_network"
sidebar_current: "docs-opc-resource-ip-network"
description: |-
  Creates and manages an IP Network
---

# opc\_compute\_ip_network

The ``opc_compute_ip_network`` resource creates and manages an IP Network.

## Example Usage

```hcl
resource "opc_compute_ip_network" "foo" {
  name                = "my-ip-network"
  description         = "my IP Network"
  ip_address_prefix   = "10.0.1.0/24"
  ip_network_exchange = "${opc_compute_ip_exchange.foo.name}"
  public_napt_enabled = false
  tags                = ["tag1", "tag2"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the IP Network.

* `ip_address_prefix` - (Required) The IPv4 address prefix, in CIDR format.

* `description` - (Optional) The description of the IP Network.

* `ip_network_exchange` - (Optional) Specify the IP Network exchange to which the IP Network belongs to.

* `public_napt_enabled` - (Optional) If true, enable public internet access using NAPT for VNICs without any public IP Reservation. Defaults to `false`.

## Attributes Reference

The following attributes are exported:

* `name` - The name of the IP Network

* `ip_address_prefix` - The IPv4 address prefix, in CIDR format.

* `description` - The description of the IP Network.

* `ip_network_exchange` - The IP Network Exchange for the IP Network

* `public_napt_enabled` - Whether public internet access using NAPT for VNICs without any public IP Reservation or not.

* `uri` - Uniform Resource Identifier for the IP Network

## Import

IP Networks can be imported using the `resource name`, e.g.

```shell
$ terraform import opc_compute_ip_network.default example
```
