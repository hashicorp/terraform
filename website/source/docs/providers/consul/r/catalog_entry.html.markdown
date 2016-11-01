---
layout: "consul"
page_title: "Consul: consul_catalog_entry"
sidebar_current: "docs-consul-resource-catalog-entry"
description: |-
  Provides access to Catalog data in Consul. This can be used to define a node or a service. Currently, defining health checks is not supported.
---

# consul\_catalog\_entry

Provides access to Catalog data in Consul. This can be used to define a node or a service. Currently, defining health checks is not supported.

## Example Usage

```
resource "consul_catalog_entry" "app" {
    address = "192.168.10.10"
    node = "foobar"
    service = {
        address = "127.0.0.1"
        id = "redis1"
        name = "redis"
        port = 8000
        tags = ["master", "v1"]
    }
}
```

## Argument Reference

The following arguments are supported:

* `address` - (Required) The address of the node being added to
  or referenced in the catalog.

* `node` - (Required) The name of the node being added to or
  referenced in the catalog.

* `service` - (Optional) A service to optionally associated with
  the node. Supported values documented below.

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
* `node` - The id of the service, defaults to the value of `name`.
