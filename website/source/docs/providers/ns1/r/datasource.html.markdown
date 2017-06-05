---
layout: "ns1"
page_title: "NS1: ns1_datasource"
sidebar_current: "docs-ns1-resource-datasource"
description: |-
  Provides a NS1 Data Source resource.
---

# ns1\_datasource

Provides a NS1 Data Source resource. This can be used to create, modify, and delete data sources.

## Example Usage

```hcl
resource "ns1_datasource" "example" {
  name       = "example"
  sourcetype = "nsone_v1"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The free form name of the data source.
* `sourcetype` - (Required) The data sources type, listed in API endpoint https://api.nsone.net/v1/data/sourcetypes.
* `config` - (Optional) The data source configuration, determined by its type.

