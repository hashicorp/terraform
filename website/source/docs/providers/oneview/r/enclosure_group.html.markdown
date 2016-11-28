---
layout: "oneview"
page_title: "Oneview: enclosure_group"
sidebar_current: "docs-oneview-enclosure-group"
description: |-
  Creates an enclosure-group.
---

# oneview\_enclosure\_group

Creates an enclosure group.

## Example Usage

```js
resource "oneview_enclosure_group" "default" {
  name = "default-enclosure-group"
  logical_interconnect_broups = ["${oneview_logical_interconnect_group.primary.name}", 
                                 "${oneview_logical_interconnect_group.secondary.name}"]
}
```

## Argument Reference

The following arguments are supported: 

* `name` - (Required) A unique name for the resource.

---

* `logical_interconnect_groups` - (Optional) The set of logical interconnect group names to associate with the enclosure group.

* `stacking mode` - (Optional) Stacking mode of the enclosure group. Defaults to Enclosure.

* `number_of_bays` - (Optional) The number of interconnect bay mappings. Defaults to 8.

In addition to the arguments listed above, the following computed attributes are exported:

* `uri` - The URI of the created resource.
