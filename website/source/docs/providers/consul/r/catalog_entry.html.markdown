---
layout: "consul"
page_title: "Consul: consul_catalog_entry"
sidebar_current: "docs-consul-resource-catalog-entry"
description: |-
  Registers a node or service with the Consul Catalog.  Currently, defining health checks is not supported.
---

# consul_catalog_entry

Registers a node or service with the [Consul Catalog](https://www.consul.io/docs/agent/http/catalog.html#catalog_register).
Currently, defining health checks is not supported.

## Example Usage

```hcl
resource "consul_catalog_entry" "app" {
  address = "192.168.10.10"
  node    = "foobar"

  service = {
    address = "127.0.0.1"
    id      = "redis1"
    name    = "redis"
    port    = 8000
    tags    = ["master", "v1"]
  }
}
```

## Argument Reference

The following arguments are supported:

* `address` - (Required) The address of the node being added to,
  or referenced in the catalog.

* `node` - (Required) The name of the node being added to, or
  referenced in the catalog.

* `service` - (Optional) A service to optionally associated with
  the node. Supported values are documented below.

* `datacenter` - (Optional) The datacenter to use. This overrides the
  datacenter in the provider setup and the agent's default datacenter.

* `token` - (Optional) ACL token.

The `service` block supports the following:

* `address` - (Optional) The address of the service. Defaults to the
  node address.
* `id` - (Optional) The ID of the service. Defaults to the `name`.
* `name` - (Required) The name of the service
* `port` - (Optional) The port of the service.
* `tags` - (Optional) A list of values that are opaque to Consul,
  but can be used to distinguish between services or nodes.

## Attributes Reference

The following attributes are exported:

* `address` - The address of the service.
* `node` - The ID of the service, defaults to the value of `name`.
