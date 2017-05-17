---
layout: "pagerduty"
page_title: "PagerDuty: pagerduty_service"
sidebar_current: "docs-pagerduty-resource-service"
description: |-
  Creates and manages a service in PagerDuty.
---

# pagerduty\_service

A [service](https://v2.developer.pagerduty.com/v2/page/api-reference#!/Services/get_services) represents something you monitor (like a web service, email service, or database service). It is a container for related incidents that associates them with escalation policies.


## Example Usage

```hcl
resource "pagerduty_user" "example" {
  name  = "Earline Greenholt"
  email = "125.greenholt.earline@graham.name"
  teams = ["${pagerduty_team.example.id}"]
}

resource "pagerduty_escalation_policy" "foo" {
  name      = "Engineering Escalation Policy"
  num_loops = 2

  rule {
    escalation_delay_in_minutes = 10

    target {
      type = "user"
      id   = "${pagerduty_user.example.id}"
    }
  }
}

resource "pagerduty_service" "example" {
  name                    = "My Web App"
  auto_resolve_timeout    = 14400
  acknowledgement_timeout = 600
  escalation_policy       = "${pagerduty_escalation_policy.example.id}"
}
```

## Argument Reference

The following arguments are supported:

  * `name` - (Required) The name of the service.
  * `description` - (Optional) A human-friendly description of the escalation policy.
    If not set, a placeholder of "Managed by Terraform" will be set.
  * `auto_resolve_timeout` - (Optional) Time in seconds that an incident is automatically resolved if left open for that long. Disabled if not set.
  * `acknowledgement_timeout` - (Optional) Time in seconds that an incident changes to the Triggered State after being Acknowledged. Disabled if not set.
  * `escalation_policy` - (Required) The escalation policy used by this service.

You may specify one optional `incident_urgency_rule` block configuring what urgencies to use.
Your PagerDuty account must have the `urgencies` ability to assign an incident urgency rule.
The block contains the following arguments.

  * `type` - The type of incident urgency: `constant` or `use_support_hours` (when depending on specific suppor hours; see `support_hours`).
  * `during_support_hours` - (Optional) Incidents' urgency during support hours.
  * `outside_support_hours` - (Optional) Incidents' urgency outside of support hours.

When using `type = "use_support_hours"` in `incident_urgency_rule` you have to specify exactly one otherwise optional `support_hours` block.
Changes to `support_hours` necessitate re-creating the service resource. Account must have the `service_support_hours` ability to assign support hours.
The block contains the following arguments.

  * `type` - The type of support hours. Can be `fixed_time_per_day`.
  * `time_zone` - The time zone for the support hours.
  * `days_of_week` - Array of days of week as integers.
  * `start_time` - The support hours' starting time of day.
  * `end_time` - The support hours' ending time of day.

When using `type = "use_support_hours"` in the `incident_urgency_rule` block you have to also specify `scheduled_actions` for the service. Otherwise `scheduled_actions` is optional. Changes necessitate re-createing the service resource.

  * `type` - The type of scheduled action. Currently, this must be set to `urgency_change`.
  * `at` - Represents when scheduled action will occur.
  * `name` - Designates either the start or the end of the scheduled action. Can be `support_hours_start` or `support_hours_end`.

Below is an example for a `pagerduty_service` resource with `incident_urgency_rules` with `type = "use_support_hours"`, `support_hours` and a default `scheduled_action` as well.

```hcl
resource "pagerduty_service" "foo" {
  name                    = "bar"
  description             = "bar bar bar"
  auto_resolve_timeout    = 3600
  acknowledgement_timeout = 3600
  escalation_policy       = "${pagerduty_escalation_policy.foo.id}"

  incident_urgency_rule {
    type = "use_support_hours"

    during_support_hours {
      type    = "constant"
      urgency = "high"
    }

    outside_support_hours {
      type    = "constant"
      urgency = "low"
    }
  }

  support_hours {
    type         = "fixed_time_per_day"
    time_zone    = "America/Lima"
    start_time   = "09:00:00"
    end_time     = "17:00:00"
    days_of_week = [1, 2, 3, 4, 5]
  }

  scheduled_actions {
    type       = "urgency_change"
    to_urgency = "high"

    at {
      type = "named_time"
      name = "support_hours_start"
    }
  }
}
```

## Attributes Reference

The following attributes are exported:

  * `id` - The ID of the service.

## Import

Services can be imported using the `id`, e.g.

```
$ terraform import pagerduty_service.main PLBP09X
```
