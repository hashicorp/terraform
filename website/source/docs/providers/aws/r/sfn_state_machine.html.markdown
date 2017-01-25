---
layout: "aws"
page_title: "AWS: sfn_state_machine"
sidebar_current: "docs-aws-resource-sfn-state-machine"
description: |-
  Provides a Step Function State Machine resource.
---

# sfn\_state\_machine

Provides a Step Function State Machine resource

## Example Usage

```
resource "aws_sfn_state_machine" "sfn_state_machine" {
  name = "user-updates-topic"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the state machine
* `definition` - (Required) The Amazon States Language definition of the state machine.
* `role_arn` - (Required) The Amazon Resource Name (ARN) of the IAM role to use for this state machine.

## Attributes Reference

The following attributes are exported:

* `id` - The ARN of the state machine
* `creation_date` - The date the state machine was created.
* `status` - The current status of the state machine. Either "ACTIVE" or "DELETING"

