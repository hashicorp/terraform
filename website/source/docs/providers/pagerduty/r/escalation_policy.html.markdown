---
layout: "pagerduty"
page_title: "PagerDuty: pagerduty_escalation_policy"
sidebar_current: "docs-pagerduty-resource-escalation_policy"
description: |-
  Creates and manages an escalation policy in PagerDuty.
---

# pagerduty\_escalation_policy

An [escalation policy](https://v2.developer.pagerduty.com/v2/page/api-reference#!/Escalation_Policies/get_escalation_policies) determines what user or schedule will be notified first, second, and so on when an incident is triggered. Escalation policies are used by one or more services.


## Example Usage

```hcl
resource "pagerduty_team" "example" {
  name        = "Engineering"
  description = "All engineering"
}

resource "pagerduty_user" "example" {
  name  = "Earline Greenholt"
  email = "125.greenholt.earline@graham.name"
  teams = ["${pagerduty_team.example.id}"]
}

resource "pagerduty_escalation_policy" "example" {
  name      = "Engineering Escalation Policy"
  num_loops = 2
  teams     = ["${pagerduty_team.example.id}"]

  rule {
    escalation_delay_in_minutes = 10

    target {
      type = "user"
      id   = "${pagerduty_user.example.id}"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the escalation policy.
* `teams` - (Optional) Teams associated with the policy. Account must have the `teams` ability to use this parameter.
* `description` - (Optional) A human-friendly description of the escalation policy.
  If not set, a placeholder of "Managed by Terraform" will be set.
* `num_loops` - (Optional) The number of times the escalation policy will repeat after reaching the end of its escalation.
* `rule` - (Required) An Escalation rule block. Escalation rules documented below.


Escalation rules (`rule`) supports the following:

  * `escalation_delay_in_minutes` - (Required) The number of minutes before an unacknowledged incident escalates away from this rule.
  * `targets` - (Required) A target block. Target blocks documented below.


Targets (`target`) supports the following:

  * `type` - (Optional) Can be `user`, `schedule`, `user_reference` or `schedule_reference`. Defaults to `user_reference`
  * `id` - (Required) A target ID

## Attributes Reference

The following attributes are exported:

  * `id` - The ID of the escalation policy.

## Import

Escalation policies can be imported using the `id`, e.g.

```
$ terraform import pagerduty_escalation_policy.main PLBP09X
```
