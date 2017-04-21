---
layout: "alicloud"
page_title: "Alicloud: alicloud_ess_schedule"
sidebar_current: "docs-alicloud-resource-ess-schedule"
description: |-
  Provides a ESS schedule resource.
---

# alicloud\_ess\_schedule

Provides a ESS schedule resource.

## Example Usage

```
resource "alicloud_ess_scaling_group" "scaling" {
  # Other parameters...
}

resource "alicloud_ess_scaling_configuration" "config" {
  # Other parameters...
}

resource "alicloud_ess_scaling_rule" "rule" {
  # Other parameters...
}

resource "alicloud_ess_schedule" "schedule" {
  scheduled_action    = "${alicloud_ess_scaling_rule.rule.ari}"
  launch_time         = "2017-04-29T07:30Z"
  scheduled_task_name = "sg-schedule"
}
```

## Argument Reference

The following arguments are supported:

* `scheduled_action` - (Required) Operations performed when the scheduled task is triggered. Fill in the unique identifier of the scaling rule.
* `launch_time` - (Required) Operations performed when the scheduled task is triggered. Fill in the unique identifier of the scaling rule.
* `scheduled_task_name` - (Optional) Display name of the scheduled task, which must be 2-40 characters (English or Chinese) long.
* `description` - (Optional) Description of the scheduled task, which is 2-200 characters (English or Chinese) long.
* `launch_expiration_time` - (Optional) Time period within which the failed scheduled task is retried. The default value is 600s. Value range: [0, 21600]
* `recurrence_type` - (Optional) Type of the scheduled task to be repeated. RecurrenceType, RecurrenceValue and RecurrenceEndTime must be specified. Optional values:
    - Daily: Recurrence interval by day for a scheduled task.
    - Weekly: Recurrence interval by week for a scheduled task.
    - Monthly: Recurrence interval by month for a scheduled task.
* `recurrence_value` - (Optional) Value of the scheduled task to be repeated. RecurrenceType, RecurrenceValue and RecurrenceEndTime must be specified.
    - Daily: Only one value in the range [1,31] can be filled.
    - Weekly: Multiple values can be filled. The values of Sunday to Saturday are 0 to 6 in sequence. Multiple values shall be separated by a comma “,”.
    - Monthly: In the format of A-B. The value range of A and B is 1 to 31, and the B value must be greater than the A value.
* `recurrence_end_time` - (Optional) End time of the scheduled task to be repeated. The date format follows the ISO8601 standard and uses UTC time. It is in the format of YYYY-MM-DDThh:mmZ. A time point 90 days after creation or modification cannot be entered. RecurrenceType, RecurrenceValue and RecurrenceEndTime must be specified.                                  
* `task_enabled` - (Optional) Whether to enable the scheduled task. The default value is true.
                                  
                                 
## Attributes Reference

The following attributes are exported:

* `id` - The schedule task ID.
* `scheduled_action` - The action of schedule task.
* `launch_time` - The time of schedule task be triggered.
* `scheduled_task_name` - The name of schedule task.
* `description` - The description of schedule task.
* `task_enabled` - Wether the task is enabled.