---
layout: "shield"
page_title: "Shield: shield_retention_policy"
sidebar_current: "docs-shield-resource-retention-policy"
description: |-
  Manages a retention policy in Shield.
---

# shield\_target

Manages a retention policy in Shield.

A retention policy defines how long a backup gets stored.

## Example Usage

Registering a target:

```
resource "shield_retention_policy" "test_retention" {
  name = "Test Retention"
  summary = "Terraform Test Retention"
  expires = 86400
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the target.

* `summary` - (Optional) A summary of the target.

* `expires` - (Required) The amount of seconds to keep a backup.
