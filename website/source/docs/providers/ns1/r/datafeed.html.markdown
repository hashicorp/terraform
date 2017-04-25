---
layout: "ns1"
page_title: "NS1: ns1_datafeed"
sidebar_current: "docs-ns1-resource-datafeed"
description: |-
  Provides a NS1 Data Feed resource.
---

# ns1\_datafeed

Provides a NS1 Data Feed resource. This can be used to create, modify, and delete data feeds.

## Example Usage

```hcl
resource "ns1_datasource" "example" {
  name       = "example"
  sourcetype = "nsone_v1"
}

resource "ns1_datafeed" "uswest_feed" {
  name      = "uswest_feed"
  source_id = "${ns1_datasource.example.id}"

  config = {
    label = "uswest"
  }
}

resource "ns1_datafeed" "useast_feed" {
  name      = "useast_feed"
  source_id = "${ns1_datasource.example.id}"

  config = {
    label = "useast"
  }
}
```

## Argument Reference

The following arguments are supported:

* `source_id` - (Required) The data source id that this feed is connected to.
* `name` - (Required) The free form name of the data feed.
* `config` - (Optional) The feeds configuration matching the specification in 'feed\_config' from /data/sourcetypes.

