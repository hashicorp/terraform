---
layout: "circonus"
page_title: "Circonus: broker"
sidebar_current: "docs-circonus-datasource-broker"
description: |-
    Provides details about a specific Circonus Broker.
---

# circonus_broker

`circonus_broker` provides
[details](https://login.circonus.com/resources/api/calls/broker) about a specific
[Circonus Broker](https://login.circonus.com/user/docs/Administration/Brokers).

As well as validating a given Circonus ID, this resource can be used to discover
the additional details about a broker configured within the provider.

~> **NOTE regarding `cirocnus_broker`:** The `circonus_broker` data source
actually queries and operats at the broker group level and can return more than
one broker per Circonus ID.  Details of each individual broker in the broker
group can be found via the `details` attribute described below.

## Example Usage

The following example shows how the resource might be used to obtain
the name of the Circonus Broker configured on the provider.

```
data "circonus_broker" "ashburn" {
  cid = "/broker/1"
}
```

## Argument Reference

The arguments of this data source act as filters for querying the available
regions. The given filters must match exactly one region whose data will be
exported as attributes.

* `cid` - (Optional) The Circonus ID of a given broker.

At least one of the above attributes should be provided when searching for a
broker.

## Attributes Reference

The following attributes are exported:

* `cid` - The Circonus ID of the selected Broker.

* `details` - A list of details about the individual Broker instances that make
  up the broker group.  See below for a list of attributes within each broker.

* `latitude` - The latitude of the selected Broker.

* `longitude` - The longitude of the selected Broker.

* `name` - The name of the selected Broker.

* `tags` - A list of tags assigned to the selected Broker.

* `type` - The of the selected Broker.  This value is either `circonus` for a
  Circonus-managed, public Broker, or `enterprise` for a private broker that is
  private to an account.

## Broker Details

* `cn` - The CN of an individual Broker in the Broker Group.
 
* `external_host` - The external host information for an individual Broker in
  the Broker Group.  This is useful or important when talking with a Broker
  through a NAT'ing firewall.

* `external_port` - The external port number for an individual Broker in the
  Broker Group.  This is useful or important when talking with a Broker through
  a NAT'ing firewall.

* `ip` - The IP address of an individual Broker in the Broker Group.  This is
  the IP address of the interface listening on the network.

* `min_version` - ??

* `modules` - A list of what modules (types of checks) this broker supports.

* `port` - The port the broker responds to the Circonus HTTPS REST wire protocol
  on.

* `skew` - The clock drift between this broker and the Circonus server.

* `status` - The status of this particular broker. A string containing either
  `active`, `unprovisioned`, `pending`, `provisioned`, or `retired`.

* `version` - The version of the broker software the broker is running.
