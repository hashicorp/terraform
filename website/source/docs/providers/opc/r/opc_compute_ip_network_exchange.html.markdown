---
layout: "opc"
page_title: "Oracle: opc_compute_ip_network_exchange"
sidebar_current: "docs-opc-resource-ip-network-exchange"
description: |-
  Creates and manages an IP network exchange in an OPC identity domain.
---

# opc\_compute\_ip\_network\_exchange

The ``opc_compute_ip_network_exchange`` resource creates and manages an IP network exchange in an OPC identity domain.

## Example Usage

```hcl
resource "opc_compute_ip_network_exchange" "default" {
  name = "NetworkExchange1"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the ip network exchange.

* `description` - (Optional) A description of the ip network exchange.

* `tags` - (Optional) List of tags that may be applied to the IP network exchange.

## Import

IP Network Exchange's can be imported using the `resource name`, e.g.

```shell
$ terraform import opc_compute_ip_network_exchange.exchange1 example
```
