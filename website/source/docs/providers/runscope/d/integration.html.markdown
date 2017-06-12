---
layout: "runscope"
page_title: "Runscope: runscope_integration"
sidebar_current: "docs-runscope-datasource-integration"
description: |-
  Get information about runscope integrations enabled on for your team.
---

# runscope\_integration

Use this data source to get information about a specific [integration](https://www.runscope.com/docs/api/integrations)
that you can with other runscope resources.

## Example Usage

```hcl
data "runscope_integration" "pagerduty" {
  team_uuid = "d26553c0-3537-40a8-9d3c-64b0453262a9"
  type = "pagerduty"
}

resource "runscope_environment" "environment" {
  bucket_id    = "${runscope_bucket.bucket.id}"
  name         = "test-environment"

  integrations = [
    {
      id               = "${data.runscope_integration.pagerduty.id}"
      integration_type = "pagerduty"
    }
  ]
}
```

## Argument Reference

The following arguments are supported:

* `team_uuid` - (Required) Your team unique identifier.
* `type` - (Required) Type of integration to lookup i.e. pagerduty

## Attributes Reference
* `id` - The unique identifier of the found integration.
