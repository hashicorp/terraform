---
layout: "oneview"
page_title: "Oneview: logical-switch"
sidebar_current: "docs-oneview-logical-switch"
description: |-
  Creates a logical switch.
---

# oneview\_logical\_switch

Creates a logical switch.

## Example Usage

```js
resource "oneview_logical_switch" "default" {
  name = "test-logical-switch"
  switch_type_name = "Cisco Nexus 6xxx"
  switch_count = 1
}
```

## Argument Reference

The following arguments are supported: 

* `name` - (Required) A unique name for the resource.

* `switch_type_name` - (Required) The name of the permitted switch type. 
  If this name changes, it will recreate the resource. 

* `switch_count` - (Required) The number of switches in your logical switch group. 
  
- - -

* `location_entry_type` - (Optional) The type of the location. 
  This defaults to StackingMemberId.


## Attributes Reference

In addition to the arguments listed above, the following computed attributes are exported:

* `uri` - The URI of the created resource.

* `eTag` - Entity tag/version ID of the resource.
