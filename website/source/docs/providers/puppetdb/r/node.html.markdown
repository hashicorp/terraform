---
layout: "puppetdb"
page_title: "PuppetDB: puppetdb_node"
sidebar_current: "docs-puppetdb-resource-node"
description: |-
  Provides a PuppetDB Node resource. This can be used to manage and delete nodes on PuppetDB.
---

# puppetdb\_node

Provides a PuppetDB Node resource. This can be used to manage and delete nodes on PuppetDB.

## Example usage

```hcl
# Manage an existing PuppetDB node
resource puppetdb_node "foo" {
  certname       = "foo.example.com"
}
```

## Argument Reference

The following arguments are supported:

* `certname` - (Required) The certificate name of the node.


## Attributes Reference

The following attributes are exported:

* `deactivated` - The deactivation date of the node.
* `expired` - The expiration date of the node.
* `cached_catalog_status` - Cached catalog status of the last puppet run for the node.
* `catalog_environment` - The environment for the last received catalog.
* `facts_environment` - The environment for the last received fact set.
* `report_environment` - The environment for the last received report.
* `catalog_timestamp` - The last time a catalog was received. Timestamps are always ISO-8601 compatible date/time strings.
* `facts_timestamp` - The last time a fact set was received. Timestamps are always ISO-8601 compatible date/time strings.
* `report_timestamp` - The last time a report run was complete. Timestamps are always ISO-8601 compatible date/time strings.
* `latest_report_corrective_change` - Whether the latest report for the node included events that remediated configuration drift. This field is only populated in PE.
* `latest_report_hash` - A hash of the latest report for the node.
* `latest_report_noop` - Whether the most recent report for the node was a noop run.
* `latest_report_noop_pending` - Whether the most recent report for the node contained noop events.
* `latest_report_status` - The status of the latest report.
