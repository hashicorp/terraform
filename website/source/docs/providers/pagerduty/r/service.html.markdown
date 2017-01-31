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

```
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

## Attributes Reference

The following attributes are exported:

  * `id` - The ID of the service.

## Import

Services can be imported using the `id`, e.g.

```
$ terraform import pagerduty_service.main PLBP09X
```
