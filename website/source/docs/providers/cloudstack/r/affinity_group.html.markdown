---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_affinity_group"
sidebar_current: "docs-cloudstack-resource-affinity-group"
description: |-
  Creates an affinity group.
---

# cloudstack_affinity_group

Creates an affinity group.

## Example Usage

```hcl
resource "cloudstack_affinity_group" "default" {
  name = "test-affinity-group"
  type = "host anti-affinity"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the affinity group. Changing this
    forces a new resource to be created.

* `description` - (Optional) The description of the affinity group.

* `type` - (Required) The affinity group type. Changing this
    forces a new resource to be created.

* `project` - (Optional) The name or ID of the project to register this
    affinity group to. Changing this forces a new resource to be created.

## Attributes Reference

The following attributes are exported:

* `id` - The id of the affinity group.
* `description` - The description of the affinity group.
