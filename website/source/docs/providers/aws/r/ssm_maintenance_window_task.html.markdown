---
layout: "aws"
page_title: "AWS: aws_ssm_maintenance_window_task"
sidebar_current: "docs-aws-resource-ssm-maintenance-window-task"
description: |-
  Provides an SSM Maintenance Window Task resource
---

# aws_ssm_maintenance_window_task

Provides an SSM Maintenance Window Task resource

## Example Usage

```hcl
resource "aws_ssm_maintenance_window" "window" {
  name = "maintenance-window-%s"
  schedule = "cron(0 16 ? * TUE *)"
  duration = 3
  cutoff = 1
}

resource "aws_ssm_maintenance_window_task" "task" {
  window_id = "${aws_ssm_maintenance_window.window.id}"
  task_type = "RUN_COMMAND"
  task_arn = "AWS-RunShellScript"
  priority = 1
  service_role_arn = "arn:aws:iam::187416307283:role/service-role/AWS_Events_Invoke_Run_Command_112316643"
  max_concurrency = "2"
  max_errors = "1"
  targets {
    key = "InstanceIds"
    values = ["${aws_instance.instance.id}"]
  }
  task_parameters {
    name = "commands"
    values = ["pwd"]
  }
}

resource "aws_instance" "instance" {
  ami = "ami-4fccb37f"

  instance_type = "m1.small"
}
```

## Argument Reference

The following arguments are supported:

* `window_id` - (Required) The Id of the maintenance window to register the task with.
* `max_concurrency` - (Required) The maximum number of targets this task can be run for in parallel.
* `max_errors` - (Required) The maximum number of errors allowed before this task stops being scheduled.
* `task_type` - (Required) The type of task being registered. The only allowed value is `RUN_COMMAND`.
* `task_arn` - (Required) The ARN of the task to execute.
* `service_role_arn` - (Required) The role that should be assumed when executing the task.
* `targets` - (Required) The targets (either instances or window target ids). Instances are specified using Key=InstanceIds,Values=instanceid1,instanceid2. Window target ids are specified using Key=WindowTargetIds,Values=window target id1, window target id2.
* `priority` - (Optional) The priority of the task in the Maintenance Window, the lower the number the higher the priority. Tasks in a Maintenance Window are scheduled in priority order with tasks that have the same priority scheduled in parallel.
* `logging_info` - (Optional) A structure containing information about an Amazon S3 bucket to write instance-level logs to. Documented below.
* `task_parameters` - (Optional) A structure containing information about parameters required by the particular `task_arn`. Documented below.

`logging_info` supports the following:

* `s3_bucket_name` - (Required)
* `s3_region` - (Required)
* `s3_bucket_prefix` - (Optional)

`task_parameters` supports the following:

* `name` - (Required)
* `values` - (Required)

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the maintenance window task.
