---
layout: "datadog"
page_title: "Datadog: datadog_downtime"
sidebar_current: "docs-datadog-resource-downtime"
description: |-
  Provides a Datadog downtime resource. This can be used to create and manage downtimes.
---

# datadog_downtime

Provides a Datadog downtime resource. This can be used to create and manage Datadog downtimes.

## Example Usage

```hcl
# Create a new daily 1700-0900 Datadog downtime
resource "datadog_downtime" "foo" {
  scope = ["*"]
  start = 1483308000
  end   = 1483365600

  recurrence {
    type   = "days"
    period = 1
  }
}
```

## Argument Reference

The following arguments are supported:

* `scope` - (Required) A list of items to apply the downtime to, e.g. host:X
* `start` - (Optional) POSIX timestamp to start the downtime.
* `end` - (Optional) POSIX timestamp to end the downtime.
* `recurrence` - (Optional) A dictionary to configure the downtime to be recurring.
    * `type` - days, weeks, months, or years
    * `period` - How often to repeat as an integer. For example to repeat every 3 days, select a type of days and a period of 3.
    * `week_days` - (Optional) A list of week days to repeat on. Choose from: Mon, Tue, Wed, Thu, Fri, Sat or Sun. Only applicable when type is weeks. First letter must be capitalized.
    * `until_occurrences` - (Optional) How many times the downtime will be rescheduled. `until_occurrences` and `until_date` are mutually exclusive.
    * `until_date` - (Optional) The date at which the recurrence should end as a POSIX timestamp. `until_occurrences` and `until_date` are mutually exclusive.
* `message` - (Optional) A message to include with notifications for this downtime.

## Attributes Reference

The following attributes are exported:

* `id` - ID of the Datadog downtime

## Import

Downtimes can be imported using their numeric ID, e.g.

```
$ terraform import datadog_downtime.bytes_received_localhost 2081
```
