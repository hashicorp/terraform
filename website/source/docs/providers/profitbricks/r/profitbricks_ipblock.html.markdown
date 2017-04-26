---
layout: "profitbricks"
page_title: "ProfitBricks: profitbricks_ipblock"
sidebar_current: "docs-profitbricks-resource-ipblock"
description: |-
  Creates and manages IP Block objects.
---

# profitbricks\_ipblock

Manages a IP Blocks on ProfitBricks

## Example Usage

```hcl
resource "profitbricks_ipblock" "example" {
  location = "${profitbricks_datacenter.example.location}"
  size     = 1
}
```

##Argument reference

* `location` - (Required)
* `size` - (Required)

