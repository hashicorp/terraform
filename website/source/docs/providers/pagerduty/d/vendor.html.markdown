---
layout: "pagerduty"
page_title: "PagerDuty: pagerduty_vendor"
sidebar_current: "docs-pagerduty-datasource-vendor"
description: |-
  Get information about a vendor that you can use for a service integration (e.g Amazon Cloudwatch, Splunk, Datadog).
---

# pagerduty\_vendor

Use this data source to get information about a specific [vendor][1] that you can use for a service integration (e.g Amazon Cloudwatch, Splunk, Datadog).

## Example Usage

```
data "pagerduty_vendor" "datadog" {
  name_regex = "^Datadog$"
}

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
  name    = "Datadog Integration"
  vendor  = "${data.pagerduty_vendor.datadog.id}"
  service = "${pagerduty_service.example.id}"
  type    = "generic_events_api_inbound_integration"
}
```

## Argument Reference

The following arguments are supported:

* `name_regex` - (Required) A regex string to apply to the vendor list returned by the PagerDuty API. This regex should be very specific. If your regex matches several vendors a list of found vendors will be returned so you can tweak your regex further. The final regex string is made case insensitive.

## Attributes Reference
* `name` - The short name of the found vendor.
* `type` - The generic service type for this vendor.

[1]: https://v2.developer.pagerduty.com/v2/page/api-reference#!/Vendors/get_vendors
