---
layout: "azure"
page_title: "Azure: azure_security_group"
sidebar_current: "docs-azure-resource-security-group"
description: |-
  Creates a new network security group within the context of the specified subscription.
---

# azure\_security\_group

Creates a new network security group within the context of the specified
subscription.

## Example Usage

```
resource "azure_security_group" "web" {
    name = "webservers"
    location = "West US"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the security group. Changing this forces a
    new resource to be created.

* `label` - (Optional) The identifier for the security group. The label can be
    up to 1024 characters long. Changing this forces a new resource to be
    created (defaults to the security group name)

* `location` - (Required) The location/region where the security group is
    created. Changing this forces a new resource to be created.

## Attributes Reference

The following attributes are exported:

* `id` - The security group ID.
* `label` - The identifier for the security group.
