---
layout: "newrelic"
page_title: "New Relic: newrelic_alert_channel"
sidebar_current: "docs-newrelic-resource-alert-channel"
description: |-
  Create and manage a notification channel for alerts in New Relic.
---

# newrelic\_alert\_channel

## Example Usage

```hcl
resource "newrelic_alert_channel" "foo" {
  name = "foo"
  type = "email"

  configuration = {
    recipients              = "foo@example.com"
    include_json_attachment = "1"
  }
}
```

## Argument Reference

The following arguments are supported:

  * `name` - (Required) The name of the channel.
  * `type` - (Required) The type of channel.  One of: `campfire`, `email`, `hipchat`, `opsgenie`, `pagerduty`, `slack`, `victorops`, or `webhook`.
  * `configuration` - (Required) A map of key / value pairs with channel type specific values.

## Attributes Reference

The following attributes are exported:

  * `id` - The ID of the channel.

## Import

Alert channels can be imported using the `id`, e.g.

```
$ terraform import newrelic_alert_channel.main 12345
```
