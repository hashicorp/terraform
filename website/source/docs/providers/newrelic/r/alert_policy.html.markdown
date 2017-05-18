---
layout: "newrelic"
page_title: "New Relic: newrelic_alert_policy"
sidebar_current: "docs-newrelic-resource-alert-policy"
description: |-
  Create and manage alert policies in New Relic.
---

# newrelic\_alert\_policy

## Example Usage

```hcl
resource "newrelic_alert_policy" "foo" {
  name = "foo"
}
```

## Argument Reference

The following arguments are supported:

  * `name` - (Required) The name of the policy.
  * `incident_preference` - (Optional) The rollup strategy for the policy.  Options include: `PER_POLICY`, `PER_CONDITION`, or `PER_CONDITION_AND_TARGET`.  The default is `PER_POLICY`.

## Attributes Reference

The following attributes are exported:

  * `id` - The ID of the policy.
  * `created_at` - The time the policy was created.
  * `updated_at` - The time the policy was last updated.

## Import

Alert policies can be imported using the `id`, e.g.

```
$ terraform import newrelic_alert_policy.main 12345
```
