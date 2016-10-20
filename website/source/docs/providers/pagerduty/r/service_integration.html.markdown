---
layout: "pagerduty"
page_title: "PagerDuty: pagerduty_service_integration"
sidebar_current: "docs-pagerduty-resource-service-integration"
description: |-
  Creates and manages a service integration in PagerDuty.
---

# pagerduty\_service_integration

A [service integration](https://v2.developer.pagerduty.com/v2/page/api-reference#!/Services/post_services_id_integrations) is an integration that belongs to a service.

`Note`: A service integration `cannot` be deleted via Terraform nor the PagerDuty API so if you remove a service integration, be sure to remove it from the PagerDuty Web UI afterwards. However, if you delete the `service` attached to the `integration`, the integration will be removed.


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

resource "pagerduty_service_integration" "example" {
  name    = "Generic API Service Integration"
  type    = "generic_events_api_inbound_integration"
  service = "${pagerduty_service.example.id}"
}

data "pagerduty_vendor" "datadog" {
  name = "Datadog"
}

data "pagerduty_vendor" "cloudwatch" {
  name_regex = "Amazon CloudWatch"
}

resource "pagerduty_service_integration" "datadog" {
  name    = "${data.pagerduty_vendor.datadog.name}"
  type    = "generic_events_api_inbound_integration"
  service = "${pagerduty_service.example.id}"
  vendor  = "${data.pagerduty_vendor.datadog.id}"
}

resource "pagerduty_service_integration" "datadog" {
  name    = "${data.pagerduty_vendor.datadog.name}"
  type    = "generic_events_api_inbound_integration"
  service = "${pagerduty_service.example.id}"
  vendor  = "${data.pagerduty_vendor.datadog.id}"
}
```

## Argument Reference

The following arguments are supported:

  * `name` - (Optional) The name of the service integration.
  * `type` - (Optional) The service type. Can be `aws_cloudwatch_inbound_integration`, `cloudkick_inbound_integration`,
  `event_transformer_api_inbound_integration`,
  `generic_email_inbound_integration`,
  `generic_events_api_inbound_integration`,
  `keynote_inbound_integration`,
  `nagios_inbound_integration`,
  `pingdom_inbound_integration`,
  `sql_monitor_inbound_integration`.

    When integrating with a `vendor` this can usually be set to: `${data.pagerduty_vendor.datadog.type}`

  * `service` - (Optional) The PagerDuty service that the integration belongs to.
  * `vendor` - (Optional) The vendor that this integration integrates with, if applicable. (e.g Datadog)

## Attributes Reference

The following attributes are exported:

  * `id` - The ID of the service integration.
  * `integration_key` - This is the unique key used to route events to this integration when received via the PagerDuty Events API.
  * `integration_email` - This is the unique fully-qualified email address used for routing emails to this integration for processing.
