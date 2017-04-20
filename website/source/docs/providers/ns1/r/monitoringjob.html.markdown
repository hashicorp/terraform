---
layout: "ns1"
page_title: "NS1: ns1_monitoringjob"
sidebar_current: "docs-ns1-resource-monitoringjob"
description: |-
  Provides a NS1 Monitoring Job resource.
---

# ns1\_monitoringjob

Provides a NS1 Monitoring Job resource. This can be used to create, modify, and delete monitoring jobs.

## Example Usage

```hcl
resource "ns1_monitoringjob" "uswest_monitor" {
  name          = "uswest"
  active        = true
  regions       = ["sjc", "sin", "lga"]
  job_type      = "tcp"
  frequency     = 60
  rapid_recheck = true
  policy        = "quorum"

  config = {
    send = "HEAD / HTTP/1.0\r\n\r\n"
    port = 80
    host = "example-elb-uswest.aws.amazon.com"
  }

  rules = {
    value      = "200 OK"
    comparison = "contains"
    key        = "output"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The free-form display name for the monitoring job.
* `job_type` - (Required) The type of monitoring job to be run.
* `active` - (Required) Indicates if the job is active or temporaril.y disabled.
* `regions` - (Required) The list of region codes in which to run the monitoring job.
* `frequency` - (Required) The frequency, in seconds, at which to run the monitoring job in each region.
* `rapid_recheck` - (Required) If true, on any apparent state change, the job is quickly re-run after one second to confirm the state change before notification.
* `policy` - (Required) The policy for determining the monitor's global status based on the status of the job in all regions.
* `config` - (Required) A configuration dictionary with keys and values depending on the jobs' type.
* `notify_delay` - (Optional) The time in seconds after a failure to wait before sending a notification.
* `notify_repeat` - (Optional) The time in seconds between repeat notifications of a failed job.
* `notify_failback` - (Optional) If true, a notification is sent when a job returns to an "up" state.
* `notify_regional` - (Optional) If true, notifications are sent for any regional failure (and failback if desired), in addition to global state notifications.
* `notify_list` - (Optional) The id of the notification list to send notifications to.
* `notes` - (Optional) Freeform notes to be included in any notifications about this job.
* `rules` - (Optional) A list of rules for determining failure conditions. Job Rules are documented below.

Monitoring Job Rules (`rules`) support the following:

* `key` - (Required) The output key.
* `comparison` - (Required) The comparison to perform on the the output.
* `value` - (Required) The value to compare to.

