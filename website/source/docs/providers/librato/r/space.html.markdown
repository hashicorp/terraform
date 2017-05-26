---
layout: "librato"
page_title: "Librato: librato_space"
sidebar_current: "docs-librato-resource-space"
description: |-
  Provides a Librato Space resource. This can be used to create and manage spaces on Librato.
---

# librato\_space

Provides a Librato Space resource. This can be used to
create and manage spaces on Librato.

## Example Usage

```hcl
# Create a new Librato space
resource "librato_space" "default" {
  name = "My New Space"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the space.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the space.
* `name` - The name of the space.
