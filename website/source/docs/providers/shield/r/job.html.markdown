---
layout: "shield"
page_title: "Shield: shield_job"
sidebar_current: "docs-shield-resource-job"
description: |-
  Manages a job in Shield.
---

# shield\_job

Manages a job in Shield.

A job in Shield combines store, target, retention policy and schedule
into a logical construct.

## Example Usage

Registering a job:

```
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

* `name` - (Required) The name of the target.

* `summary` - (Optional) A summary of the target.

* `store` - (Required) The UUID of the store to use.

* `target` - (Required) The UUID of the target to use.

* `retention` - (Required) The UUID of the retention policy to use.

* `schedule` - (Required) The UUID of the schedule to use.

* `paused` - (Optional) Boolean if the job is paused.
