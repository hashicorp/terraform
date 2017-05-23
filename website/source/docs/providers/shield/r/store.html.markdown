---
layout: "shield"
page_title: "Shield: shield_store"
sidebar_current: "docs-shield-resource-store"
description: |-
  Manages a store in Shield.
---

# shield\_store

Manages a store in Shield.

A store in Shield defines where the output of a job gets stored.

## Example Usage

Registering a store:

```
resource "shield_store" "test_store" {
  name = "Test-Store"
  summary = "Terraform Test Store"
  plugin = "fs"
  endpoint = "${file("test-store.json")}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the store.

* `summary` - (Optional) A summary of the store.

* `plugin` - (Required) The plugin to use for the store.

* `endpoint` - (Required) The configuration of the plugin.
