---
layout: "runscope"
page_title: "Runscope: runscope_schedule"
sidebar_current: "docs-runscope-resource-schedule"
description: |-
  Provides a Runscope schedule resource.
---

# runscope\_schedule

A [schedule](https://www.runscope.com/docs/api/schedules) resource.
Tests can be scheduled to run on frequencies up to every minute.
One ore more schedules can be configured per test with each schedule
using a unique Test-specific or Shared [Environment](environment.html).
### Creating a schedule
```hcl
resource "runscope_schedule" "daily" {
  bucket_id      = "${runscope_bucket.bucket.id}"
  test_id        = "${runscope_test.test.id}"
  interval       = "1d"
  note           = "This is a daily schedule"
  environment_id = "${runscope_environment.environment.id}"
}

resource "runscope_test" "test" {
  bucket_id   = "${runscope_bucket.bucket.id}"
  name        = "runscope test"
  description = "This is a test test..."
}

resource "runscope_bucket" "bucket" {
  name      = "terraform-provider-test"
  team_uuid = "d038db69-b5a9-45af-80d8-3be47c37e309"
}

resource "runscope_environment" "environment" {
  bucket_id = "${runscope_bucket.bucket.id}"
  name      = "test-environment"

  initial_variables {
    var1 = "true",
    var2 = "value2"
  }
}
```

## Argument Reference

The following arguments are supported:

* `bucket_id` - (Required) The id of the bucket to associate this schedule with.
* `test_id` - (Required) The id of the test to associate this schedule with.
* `environment_id` - (Required) The id of the environment to use when running the test.
If given, creates a test specific schedule, otherwise creates a shared schedule.
* `interval` - (Required) The schedule's interval, must be one of:
 * 1m — every minute
 * 5m — every 5 minutes
 * 15m — every 15 minutes
 * 30m — every 30 minutes
 * 1h — every hour
 * 6h — every 6 hours
 * 1d — every day.
* `note` - (Optional) A human-friendly description for the schedule.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the schedule.
