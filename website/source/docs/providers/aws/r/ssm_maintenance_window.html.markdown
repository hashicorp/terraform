---
layout: "aws"
page_title: "AWS: aws_ssm_maintenance_window"
sidebar_current: "docs-aws-resource-ssm-maintenance-window"
description: |-
  Provides an SSM Maintenance Window resource
---

# aws_ssm_maintenance_window

Provides an SSM Maintenance Window resource

## Example Usage

```hcl
resource "aws_ssm_maintenance_window" "production" {
  name = "maintenance-window-application"
  schedule = "cron(0 16 ? * TUE *)"
  duration = 3
  cutoff = 1
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the maintenance window.
* `schedule` - (Required) The schedule of the Maintenance Window in the form of a [cron](https://docs.aws.amazon.com/systems-manager/latest/userguide/sysman-maintenance-cron.html) or rate expression.
* `cutoff` - (Required) The number of hours before the end of the Maintenance Window that Systems Manager stops scheduling new tasks for execution.
* `duration` - (Required) The duration of the Maintenance Window in hours.
* `allow_unassociated_targets` - (Optional) Whether targets must be registered with the Maintenance Window before tasks can be defined for those targets.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the maintenance window.
