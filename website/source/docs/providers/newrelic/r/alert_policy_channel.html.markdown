---
layout: "newrelic"
page_title: "New Relic: newrelic_alert_policy_channel"
sidebar_current: "docs-newrelic-resource-alert-policy-channel"
description: |-
  Map alert policies to alert channels in New Relic.
---

# newrelic\_alert\_policy\_channel

## Example Usage

```hcl
resource "newrelic_alert_policy" "foo" {
  name = "foo"
}

resource "newrelic_alert_channel" "foo" {
  name = "foo"
  type = "email"

  configuration = {
    recipients              = "foo@example.com"
    include_json_attachment = "1"
  }
}

resource "newrelic_alert_policy_channel" "foo" {
  policy_id  = "${newrelic_alert_policy.foo.id}"
  channel_id = "${newrelic_alert_channel.foo.id}"
}
```

## Argument Reference

The following arguments are supported:

  * `policy_id` - (Required) The ID of the policy.
  * `channel_id` - (Required) The ID of the channel.
