---
layout: "aws"
page_title: "AWS: aws_api_usage_plan"
sidebar_current: "docs-aws-resource-api-usage-plan"
description: |-
  Provides an API Gateway Usage Plan.
---

# aws\_api\_usage\_plan

Provides an API Gateway Resource.

## Example Usage

```
resource "aws_api_usage_plan" "MyUsagePlan" {
  usage_plan_name = "my-usage-plan"
}
```

## Argument Reference

The API Gateway Usage Plan argument layout is a structure composed of several sub-resources - these resources are laid out below.

### Top-Level Arguments

* `name` - (Required) The name of the usage plan.
* `description` - (Required) The description of a usage plan.
* `api_stages` - (Optional) The associated [API stages](#api-stages-arguments) of the usage plan.
* `quota_settings` - (Optional) The [quota settings](#quota-settings-arguments) of the usage plan.
* `throttle_settings` - (Optional) The [throttling limits](#throttling-settings-arguments) of the usage plan.
* `product_code` - (Optional) The [throttling limits](#throttling-settings-arguments) of the usage plan.

#### Api Stages arguments

  * `api_id` (Optional) - API Id of the associated API stage in a usage plan.
  * `stage` (Optional) - API stage name of the associated API stage in a usage plan.

#### Quota Settings Arguments

  * `limit` (Optional) - The maximum number of requests that can be made in a given time period.
  * `offset` (Optional) - The number of requests subtracted from the given limit in the initial time period.
  * `period` (Optional) - The time period in which the limit applies. Valid values are "DAY", "WEEK" or "MONTH".

#### Throttling Settings Arguments

  * `burst_limit` (Optional) - The API request burst limit, the maximum rate limit over a time ranging from one to a few seconds, depending upon whether the underlying token bucket is at its full capacity.
  * `rate_limit` (Optional) - The API request steady-state rate limit.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the API resource
* `name` - The name of the usage plan.
* `description` - The description of a usage plan.
* `api_stages` - The associated API stages of the usage plan.
* `quota_settings` - The quota of the usage plan.
* `throttle_settings` - The throttling limits of the usage plan.
* `product_code` - The AWS Markeplace product identifier to associate with the usage plan as a SaaS product on AWS Marketplace.
