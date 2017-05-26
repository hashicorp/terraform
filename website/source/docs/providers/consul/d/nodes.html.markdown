---
layout: "consul"
page_title: "Consul: consul_catalog_nodes"
sidebar_current: "docs-consul-data-source-catalog-nodes"
description: |-
  Provides a list of nodes in a given Consul datacenter.
---

# consul_catalog_nodes

The `consul_catalog_nodes` data source returns a list of Consul nodes that have
been registered with the Consul cluster in a given datacenter.  By specifying a
different datacenter in the `query_options` it is possible to retrieve a list of
nodes from a different WAN-attached Consul datacenter.

## Example Usage

```hcl
data "consul_catalog_nodes" "read-dc1-nodes" {
  query_options {
    # Optional parameter: implicitly uses the current datacenter of the agent  
    datacenter = "dc1"
  }
}

# Set the description to a whitespace delimited list of the node names
resource "example_resource" "app" {
  description = "${join(" ", formatlist("%s", data.consul_catalog_nodes.node_names))}"

  # ...
}
```

## Argument Reference

The following arguments are supported:

* `datacenter` - (Optional) The Consul datacenter to query.  Defaults to the
  same value found in `query_options` parameter specified below, or if that is
  empty, the `datacenter` value found in the Consul agent that this provider is
  configured to talk to.

* `query_options` - (Optional) See below.

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
* `node_ids` - A list of the Consul node IDs.
* `node_names` - A list of the Consul node names.
* `nodes` - A list of nodes and details about each Consul agent.  The list of
  per-node attributes is detailed below.

The following is a list of the per-node attributes contained within the `nodes`
map:

* `id` - The Node ID of the Consul agent.
* [`meta`](https://www.consul.io/docs/agent/http/catalog.html#Meta) - Node meta
  data tag information, if any.
* [`name`](https://www.consul.io/docs/agent/http/catalog.html#Node) - The name
  of the Consul node.
* [`address`](https://www.consul.io/docs/agent/http/catalog.html#Address) - The
  IP address the node is advertising to the Consul cluster.
* [`tagged_addresses`](https://www.consul.io/docs/agent/http/catalog.html#TaggedAddresses) -
  List of explicit LAN and WAN IP addresses for the agent.
