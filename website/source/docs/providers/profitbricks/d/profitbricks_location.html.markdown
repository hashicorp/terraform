---
layout: "profitbricks"
page_title: "ProfitBricks : profitbrick_location"
sidebar_current: "docs-profitbricks-datasource-location"
description: |-
  Get information on a ProfitBricks Locations
---

# profitbricks\_location

The locations data source can be used to search for and return an existing location which can then be used elsewhere in the configuration.

## Example Usage

```hcl
data "profitbricks_location" "loc1" {
  name    = "karlsruhe"
  feature = "SSD"
}
```

## Argument Reference

 * `name` - (Required) Name or part of the location name to search for.
 * `feature` - (Optional) A desired feature that the location must be able to provide.

## Attributes Reference

 * `id` - UUID of the location
