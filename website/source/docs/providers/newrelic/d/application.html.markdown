---
layout: "newrelic"
page_title: "New Relic: newrelic_application"
sidebar_current: "docs-newrelic-datasource-application"
description: |-
  Looks up the information about an application in New Relic.
---

# newrelic\_application

Use this data source to get information about a specific application in New Relic.

## Example Usage

```hcl
data "newrelic_application" "app" {
  name = "my-app"
}

resource "newrelic_alert_policy" "foo" {
  name = "foo"
}

resource "newrelic_alert_condition" "foo" {
  policy_id = "${newrelic_alert_policy.foo.id}"

  name        = "foo"
  type        = "apm_app_metric"
  entities    = ["${data.newrelic_application.app.id}"]
  metric      = "apdex"
  runbook_url = "https://www.example.com"

  term {
    duration      = 5
    operator      = "below"
    priority      = "critical"
    threshold     = "0.75"
    time_function = "all"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the application in New Relic.

## Attributes Reference
* `id` - The ID of the application.
* `instance_ids` - A list of instance IDs associated with the application.
* `host_ids` - A list of host IDs associated with the application.
