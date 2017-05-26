---
layout: "newrelic"
page_title: "New Relic: newrelic_alert_condition"
sidebar_current: "docs-newrelic-resource-alert-condition"
description: |-
  Create and manage an alert condition for a policy in New Relic.
---

# newrelic\_alert\_condition

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

  * `policy_id` - (Required) The ID of the policy where this condition should be used.
  * `name` - (Required) The title of the condition
  * `type` - (Required) The type of condition. One of: `apm_app_metric`, `apm_kt_metric`, `servers_metric`, `browser_metric`, `mobile_metric`
  * `entities` - (Required) The instance IDS associated with this condition.
  * `metric` - (Required) The metric field accepts parameters based on the `type` set.
  * `runbook_url` - (Optional) Runbook URL to display in notifications.
  * `condition_scope` - (Optional) `instance` or `application`.  This is required if you are using the JVM plugin in New Relic.
  * `term` - (Required) A list of terms for this condition. See [Terms](#terms) below for details.
  * `user_defined_metric` - (Optional) A custom metric to be evaluated.
  * `user_defined_value_function` - (Optional) One of: `average`, `min`, `max`, `total`, or `sample_size`.

## Terms

The `term` mapping supports the following arguments:

  * `duration` - (Required) In minutes, must be: `5`, `10`, `15`, `30`, `60`, or `120`.
  * `operator` - (Optional) `above`, `below`, or `equal`.  Defaults to `equal`.
  * `priority` - (Optional) `critical` or `warning`.  Defaults to `critical`.
  * `threshold` - (Required) Must be 0 or greater.
  * `time_function` - (Required) `all` or `any`.

## Attributes Reference

The following attributes are exported:

  * `id` - The ID of the alert condition.

## Import

Alert conditions can be imported using the `id`, e.g.

```
$ terraform import newrelic_alert_condition.main 12345
```
