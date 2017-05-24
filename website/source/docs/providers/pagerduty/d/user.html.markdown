---
layout: "pagerduty"
page_title: "PagerDuty: pagerduty_user"
sidebar_current: "docs-pagerduty-datasource-user"
description: |-
  Get information about a user that you can use for a service integration (e.g Amazon Cloudwatch, Splunk, Datadog).
---

# pagerduty\_user

Use this data source to get information about a specific [user][1] that you can use for other Pager Duty resources.

## Example Usage

```hcl
data "pagerduty_user" "me" {
  email = "me@example.com"
}

resource "pagerduty_escalation_policy" "foo" {
  name      = "Engineering Escalation Policy"
  num_loops = 2

  rule {
    escalation_delay_in_minutes = 10

    target {
      type = "user"
      id   = "${data.pagerduty_user.me.id}"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `email` - (Required) The email to use to find a user in the PagerDuty API.

## Attributes Reference
* `name` - The short name of the found user.

[1]: https://v2.developer.pagerduty.com/v2/page/api-reference#!/Users/get_users
