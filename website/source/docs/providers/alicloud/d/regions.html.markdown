---
layout: "alicloud"
page_title: "Alicloud: alicloud_regions"
sidebar_current: "docs-alicloud-datasource-regions"
description: |-
    Provides a list of Availability Regions which can be used by an Alicloud account.
---

# alicloud_regions

The Regions data source allows access to the list of Alicloud Regions.

## Example Usage

```hcl
data "alicloud_regions" "current" {
  current = true
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) The full name of the region to select.
* `current` - (Optional) Set to true to match only the region configured in the provider.

## Attributes Reference

The following attributes are exported:

* `id` - ID of the region.
* `local_name` - Name of the region in the local language.
