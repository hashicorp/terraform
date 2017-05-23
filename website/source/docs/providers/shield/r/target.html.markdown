---
layout: "shield"
page_title: "Shield: shield_target"
sidebar_current: "docs-shield-resource-target"
description: |-
  Manages a target in Shield.
---

# shield\_target

Manages a target in Shield.

A target in Shield defines the plugin, it’s the corresponding [configuration](https://github.com/starkandwayne/shield#plugins)
and the shield agents which will execute the job.

## Example Usage

Registering a target:

```
resource "shield_target" "test_target" {
  name = "Test-Target"
  summary = "Terraform Test Target"
  plugin = "mysql"
  endpoint = "${file("test-target.json")}"
  agent = "localhost:5444"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the target.

* `summary` - (Optional) A summary of the target.

* `plugin` - (Required) The plugin to use for the target.

* `agent` - (Required) The agent including port which will run the job.
