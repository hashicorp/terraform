---
layout: "circonus"
page_title: "Circonus: collector"
sidebar_current: "docs-circonus-datasource-collector"
description: |-
    Provides details about a specific Circonus Collector.
---

# circonus_collector

`circonus_collector` provides
[details](https://login.circonus.com/resources/api/calls/broker) about a specific
[Circonus Collector](https://login.circonus.com/user/docs/Administration/Brokers).

As well as validating a given Circonus ID, this resource can be used to discover
the additional details about a collector configured within the provider.  The
results of a `circonus_collector` API call can return more than one collector
per Circonus ID.  Details of each individual collector in the group of
collectors can be found via the `details` attribute described below.

~> **NOTE regarding `cirocnus_collector`:** The `circonus_collector` data source
actually queries and operates on Circonus "brokers" at the broker group level.
The `circonus_collector` is simply a renamed Circonus "broker" to make it clear
what the function of the "broker" actually does: act as a fan-in agent that
either pulls or has metrics pushed into it and funneled back through Circonus.

## Example Usage

The following example shows how the resource might be used to obtain
the name of the Circonus Collector configured on the provider.

```hcl
data "circonus_collector" "ashburn" {
  id = "/broker/1"
}
```

## Argument Reference

The arguments of this data source act as filters for querying the available
regions. The given filters must match exactly one region whose data will be
exported as attributes.

* `id` - (Optional) The Circonus ID of a given collector.

At least one of the above attributes should be provided when searching for a
collector.

## Attributes Reference

The following attributes are exported:

* `id` - The Circonus ID of the selected Collector.

* `details` - A list of details about the individual Collector instances that
  make up the group of collectors.  See below for a list of attributes within
  each collector.

* `latitude` - The latitude of the selected Collector.

* `longitude` - The longitude of the selected Collector.

* `name` - The name of the selected Collector.

* `tags` - A list of tags assigned to the selected Collector.

* `type` - The of the selected Collector.  This value is either `circonus` for a
  Circonus-managed, public Collector, or `enterprise` for a private collector that is
  private to an account.

## Collector Details

* `cn` - The CN of an individual Collector in the Collector Group.

* `external_host` - The external host information for an individual Collector in
  the Collector Group.  This is useful or important when talking with a Collector
  through a NAT'ing firewall.

* `external_port` - The external port number for an individual Collector in the
  Collector Group.  This is useful or important when talking with a Collector through
  a NAT'ing firewall.

* `ip` - The IP address of an individual Collector in the Collector Group.  This is
  the IP address of the interface listening on the network.

* `min_version` - ??

* `modules` - A list of what modules (types of checks) this collector supports.

* `port` - The port the collector responds to the Circonus HTTPS REST wire protocol
  on.

* `skew` - The clock drift between this collector and the Circonus server.

* `status` - The status of this particular collector. A string containing either
  `active`, `unprovisioned`, `pending`, `provisioned`, or `retired`.

* `version` - The version of the collector software the collector is running.
