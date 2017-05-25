---
layout: "clc"
page_title: "clc: clc_group"
sidebar_current: "docs-clc-resource-group"
description: |-
  Manages a CLC server group.
---

# clc_group

Manages a CLC server group. Either provisions or resolves to an existing group.

See also [Complete API documentation](https://www.ctl.io/api-docs/v2/#groups).

## Example Usage

```hcl
# Provision/Resolve a server group
resource "clc_group" "frontends" {
  location_id = "WA1"
  name        = "frontends"
  parent      = "Default Group"
}

output "group_id" {
  value = "clc_group.frontends.id"
}
```


## Argument Reference


The following arguments are supported:

* `name` - (Required, string) The name (or GUID) of this server group. Will resolve to existing if present.
* `parent` - (Required, string) The name or ID of the parent group. Will error if absent or unable to resolve.
* `location_id` - (Required, string) The datacenter location of both parent group and this group.
   Examples: "WA1", "VA1"
* `description` - (Optional, string) Description for server group (visible in control portal only)
* `custom_fields` - (Optional) See [CustomFields](#custom_fields) below for details.



<a id="custom_fields"></a>
## CustomFields

`custom_fields` is a block within the configuration that may be
repeated to bind custom fields for a server. CustomFields need be set
up in advance. Each `custom_fields` block supports the following:

* `id` - (Required, string) The ID of the custom field to set.
* `value` - (Required, string) The value for the specified field.
