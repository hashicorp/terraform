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

```
resource "pagerduty_user" "example" {
    name  = "Earline Greenholt"
    email = "125.greenholt.earline@graham.name"
    teams = ["${pagerduty_team.example.id}"]
}

resource "pagerduty_escalation_policy" "example" {
  name             = "Engineering"
  description      = "Engineering Escalation Policy"
  num_loops        = 2
  escalation_rules = <<EOF
  [
      {
          "escalation_delay_in_minutes": 10,
          "targets": [
              {
                  "type": "user",
                  "id": "${pagerduty_user.example.id}"
              }
          ]
      }
  ]
EOF
}
```

## Argument Reference

The following arguments are supported:

  * `name` - (Required) The name of the escalation policy.
  * `description` - (Optional) A human-friendly description of the escalation policy.
    If not set, a placeholder of "Managed by Terraform" will be set.
  * `num_loops` (Optional) The number of times the escalation policy will repeat after reaching the end of its escalation.
  * `escalation_rules` (Required) A JSON array containing escalation rules. Each rule must have `escalation_delay_in_minutes` defined as well as an array containing `targets`
    * `escalation_delay_in_minutes` (Required) The number of minutes before an unacknowledged incident escalates away from this rule.
    * `targets` (Required) The targets an incident should be assigned to upon reaching this rule.

## Attributes Reference

The following attributes are exported:

  * `id` - The ID of the escalation policy.
  * `name` - The name of the escalation policy.
  * `description` - Escalation policy description.
  * `num_loops` - The number of times the escalation policy will repeat after reaching the end of its escalation.
