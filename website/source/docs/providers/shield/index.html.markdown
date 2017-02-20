---
layout: "shield"
page_title: "Provider: Shield"
sidebar_current: "docs-shield-index"
description: |-
  Shield is a standalone system that can perform backup and restore functions for a wide variety of pluggable data systems.
---

# Shield Provider

[Shield](https://github.com/starkandwayne/shield) is a standalone system that can perform backup and restore functions for a wide variety of pluggable data systems . The Shield
provider exposes resources to interact with a Shield server.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the Nomad provider
provider "shield" {
  serverurl = "shield.example.com"
  username = "admin"
  password = "fancypassword"
  insecure = false
}

# Register a target
resource "shield_target" "test_target" {
  name = "Test-Target"
  summary = "Terraform Test Target"
  plugin = "mysql"
  endpoint = "${file("test-target.json")}"
  agent = "127.0.0.1:5444"
}

# Register a schedule
resource "shield_schedule" "test_schedule" {
  name = "Test-Schedule"
  summary = "Terraform Test Schedule"
  when = "daily 1am"
}

# Register a store
resource "shield_store" "test_store" {
  name = "Test-Store"
  summary = "Terraform Test Store"
  plugin = "fs"
  endpoint = "${file("test-store.json")}"
}

# Register a retention policy
resource "shield_retention_policy" "test_retention" {
  name = "Test Retention"
  summary = "Terraform Test Retention"
  expires = 86400
}

# Register a job
resource "shield_job" "test_job" {
  name = "Test-Job"
  summary = "Terraform Test Job"
  store = "${ shield_store.test_store.uuid }"
  target = "${ shield_target.test_target.uuid }"
  retention = "${ shield_retention_policy.test_retention.uuid }"
  schedule = "${ shield_schedule.test_schedule.uuid }"
  paused = false
}
```

## Argument Reference

The following arguments are supported:

* `serverurl` - (Required) The HTTPS endpoint of the shield daemon.
* `username` - (Required) Username of the shield user.
* `password` - (Required) Password of the shield user.
* `insecure` - (Optional) Ignores certificate warnings (eg. allows self-signed certificates)
