---
layout: "profitbricks"
page_title: "ProfitBricks: profitbricks_datacenter"
sidebar_current: "docs-profitbricks-resource-datacenter"
description: |-
  Creates and manages Profitbricks Virtual Data Center.
---

# profitbricks\_datacenter

Manages a Virtual Data Center on ProfitBricks

## Example Usage

```hcl
resource "profitbricks_datacenter" "example" {
  name        = "datacenter name"
  location    = "us/las"
  description = "datacenter description"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required)[string] The name of the Virtual Data Center.
* `location` - (Required)[string] The physical location where the data center will be created.
* `description` - (Optional)[string] Description for the data center.
