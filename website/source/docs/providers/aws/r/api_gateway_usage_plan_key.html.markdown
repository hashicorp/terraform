---
layout: "aws"
page_title: "AWS: aws_api_gateway_usage_plan_key"
sidebar_current: "docs-aws-resource-api-gateway-usage-plan-key"
description: |-
  Provides an API Gateway Usage Plan Key.
---

# aws_api_gateway_usage_plan_key

Provides an API Gateway Usage Plan Key.

## Example Usage

```hcl
resource "aws_api_gateway_rest_api" "test" {
  name = "MyDemoAPI"
}

...

resource "aws_api_gateway_usage_plan" "myusageplan" {
  name = "my_usage_plan"
}

resource "aws_api_gateway_api_key" "mykey" {
  name = "my_key"

  stage_key {
    rest_api_id = "${aws_api_gateway_rest_api.test.id}"
    stage_name  = "${aws_api_gateway_deployment.foo.stage_name}"
  }
}

resource "aws_api_gateway_usage_plan_key" "main" {
  key_id        = "${aws_api_gateway_api_key.mykey.id}"
  key_type      = "API_KEY"
  usage_plan_id = "${aws_api_gateway_usage_plan.myusageplan.id}"
}
```

## Argument Reference

The following arguments are supported:

* `key_id` - (Required) The identifier of the API key resource.
* `key_type` - (Required) The type of the API key resource. Currently, the valid key type is API_KEY.
* `usage_plan_id` - (Required) The Id of the usage plan resource representing to associate the key to.

## Attributes Reference

The following attributes are exported:

* `id` - The Id of a usage plan key.
* `key_id` - The type of a usage plan key. Currently, the valid key type is API_KEY.
* `key_type` - The ID of the API resource
* `usage_plan_id` - The ID of the API resource
* `name` - The name of a usage plan key.
* `value` - The value of a usage plan key.
