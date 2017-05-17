---
layout: "consul"
page_title: "Consul: consul_catalog_service"
sidebar_current: "docs-consul-data-source-catalog-service"
description: |-
  Provides details about a specific Consul service
---

# consul_catalog_service

`consul_catalog_service` provides details about a specific Consul service in a
given datacenter.  The results include a list of nodes advertising the specified
service, the node's IP address, port number, node ID, etc.  By specifying a
different datacenter in the `query_options` it is possible to retrieve a list of
services from a different WAN-attached Consul datacenter.

This data source is different from the `consul_catalog_services` (plural) data
source, which provides a summary of the current Consul services.

## Example Usage

```hcl
data "consul_catalog_service" "read-consul-dc1" {
  query_options {
    # Optional parameter: implicitly uses the current datacenter of the agent  
    datacenter = "dc1"
  }

  name = "consul"
}

# Set the description to a whitespace delimited list of the node names
resource "example_resource" "app" {
  description = "${join(" ", data.consul_catalog_service.nodes)}"

  # ...
}
```

## Argument Reference

The following arguments are supported:

* `datacenter` - (Optional) The Consul datacenter to query.  Defaults to the
  same value found in `query_options` parameter specified below, or if that is
  empty, the `datacenter` value found in the Consul agent that this provider is
  configured to talk to.

* `name` - (Required) The service name to select.

* `query_options` - (Optional) See below.

* `tag` - (Optional) A single tag that can be used to filter the list of nodes
  to return based on a single matching tag..

The `query_options` block supports the following:

* `allow_stale` - (Optional) When `true`, the default, allow responses from
  Consul servers that are followers.

* `require_consistent` - (Optional) When `true` force the client to perform a
  read on at least quorum servers and verify the result is the same.  Defaults
  to `false`.

* `token` - (Optional) Specify the Consul ACL token to use when performing the
  request.  This defaults to the same API token configured by the `consul`
  provider but may be overriden if necessary.

* `wait_index` - (Optional) Index number used to enable blocking quereis.

* `wait_time` - (Optional) Max time the client should wait for a blocking query
  to return.

## Attributes Reference

The following attributes are exported:

* `datacenter` - The datacenter the keys are being read from to.
* `name` - The name of the service
* `tag` - The name of the tag used to filter the list of nodes in `service`.
* `service` - A list of nodes and details about each endpoint advertising a
  service.  Each element in the list is a map of attributes that correspond to
  each individual node.  The list of per-node attributes is detailed below.

The following is a list of the per-node `service` attributes:

* [`create_index`](https://www.consul.io/docs/agent/http/catalog.html#CreateIndex) -
  The index entry at which point this entry was added to the catalog.
* [`modify_index`](https://www.consul.io/docs/agent/http/catalog.html#ModifyIndex) -
  The index entry at which point this entry was modified in the catalog.
* [`node_address`](https://www.consul.io/docs/agent/http/catalog.html#Address) -
  The address of the Consul node advertising the service.
* `node_id` - The Node ID of the Consul agent advertising the service.
* [`node_meta`](https://www.consul.io/docs/agent/http/catalog.html#Meta) - Node
  meta data tag information, if any.
* [`node_name`](https://www.consul.io/docs/agent/http/catalog.html#Node) - The
  name of the Consul node.
* [`address`](https://www.consul.io/docs/agent/http/catalog.html#ServiceAddress) -
  The IP address of the service.  If the `ServiceAddress` in the Consul catalog
  is empty, this value is automatically populated with the `node_address` (the
  `Address` in the Consul Catalog).
* [`enable_tag_override`](https://www.consul.io/docs/agent/http/catalog.html#ServiceEnableTagOverride) -
  Whether service tags can be overridden on this service.
* [`id`](https://www.consul.io/docs/agent/http/catalog.html#ServiceID) - A
  unique service instance identifier.
* [`name`](https://www.consul.io/docs/agent/http/catalog.html#ServiceName) - The
  name of the service.
* [`port`](https://www.consul.io/docs/agent/http/catalog.html#ServicePort) -
  Port number of the service.
* [`tagged_addresses`](https://www.consul.io/docs/agent/http/catalog.html#TaggedAddresses) -
  List of explicit LAN and WAN IP addresses for the agent.
* [`tags`](https://www.consul.io/docs/agent/http/catalog.html#ServiceTags) -
  List of tags for the service.
