---
layout: "pagerduty"
page_title: "PagerDuty: pagerduty_on_call"
sidebar_current: "docs-pagerduty-datasource-on_call"
description: |-
  Get information about who's on call.
---

# pagerduty\_on_call

Use this data source to get all of the users [on call][1] in a given schedule.

## Example Usage

```
resource "pagerduty_schedule" "foo" {
  name      = "Daily Engineering Rotation"
  time_zone = "America/New_York"

  layer {
    name                         = "Night Shift"
    start                        = "2015-11-06T20:00:00-05:00"
    rotation_virtual_start       = "2015-11-06T20:00:00-05:00"
    rotation_turn_length_seconds = 86400
    users                        = ["${pagerduty_user.foo.id}"]

    restriction {
      type              = "daily_restriction"
      start_time_of_day = "08:00:00"
      duration_seconds  = 32400
    }
  }
}

data "pagerduty_on_call" "on_call" {}

resource "pagerduty_team" "on_call" {
  name        = "On call"
  description = "Primarily used by ${data.pagerduty_on_call.oncalls.0.id}"
}
```

## Argument Reference

The following arguments are supported:

* `time_zone`              - (Optional) Time zone in which dates in the result will be rendered.
* `include`                - (Optional) List of of additional details to include. Can be `escalation_policies`, `users`, `schedules`.
* `user_ids`               - (Optional) Filters the results, showing only on-calls for the specified user IDs.
* `escalation_policy_ids`  - (Optional) Filters the results, showing only on-calls for the specified escalation policy IDs.
* `user_ids`               - (Optional) Filters the results, showing only on-calls for the specified schedule IDs.
* `since`                  - (Optional) The start of the time range over which you want to search. If an on-call period overlaps with the range, it will be included in the result. Defaults to current time. The search range cannot exceed 3 months.
* `until`                  - (Optional) The end of the time range over which you want to search. If an on-call period overlaps with the range, it will be included in the result. Defaults to current time. The search range cannot exceed 3 months, and the until time cannot be before the since time.
* `earliest`                  - (Optional) This will filter on-calls such that only the earliest on-call for each combination of escalation policy, escalation level, and user is returned. This is useful for determining when the "next" on-calls are for a given set of filters.

## Attributes Reference
* `oncalls` - A list of on-call entries during a given time range.

[1]: https://v2.developer.pagerduty.com/v2/page/api-reference#!/On-Calls/get_oncalls
