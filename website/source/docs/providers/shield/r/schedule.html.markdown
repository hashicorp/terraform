---
layout: "shield"
page_title: "Shield: shield_schedule"
sidebar_current: "docs-shield-resource-schedule"
description: |-
  Manages a schedule in Shield.
---

# shield\_schedule

Manages a schedule in Shield.

A schedule in Shield defines when a job gets executed.

## Example Usage

Registering a schedule:

```
resource "shield_schedule" "test_schedule" {
  name = "Test-Schedule"
  summary = "Terraform Test Schedule"
  when = "daily 1am"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the schedule.

* `summary` - (Optional) A summary of the schedule.

* `when` - (Required) The declaration when a job gets executed.
