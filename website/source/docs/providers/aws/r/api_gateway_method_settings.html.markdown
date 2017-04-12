---
layout: "aws"
page_title: "AWS: aws_api_gateway_method_settings"
sidebar_current: "docs-aws-resource-api-gateway-method-settings"
description: |-
  Provides an API Gateway Method Settings, e.g. logging or monitoring.
---

# aws\_api\_gateway\_method\_settings

Provides an API Gateway Method Settings, e.g. logging or monitoring.

## Example Usage

```hcl
resource "aws_api_gateway_method_settings" "s" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  stage_name  = "${aws_api_gateway_stage.test.stage_name}"
  method_path = "${aws_api_gateway_resource.test.path_part}/${aws_api_gateway_method.test.http_method}"

  settings {
    metrics_enabled = true
    logging_level   = "INFO"
  }
}

resource "aws_api_gateway_rest_api" "test" {
  name = "MyDemoAPI"
  description = "This is my API for demonstration purposes"
}

resource "aws_api_gateway_deployment" "test" {
  depends_on = ["aws_api_gateway_integration.test"]
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  stage_name = "dev"
}

resource "aws_api_gateway_stage" "test" {
  stage_name = "prod"
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  deployment_id = "${aws_api_gateway_deployment.test.id}"
}

resource "aws_api_gateway_resource" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  parent_id   = "${aws_api_gateway_rest_api.test.root_resource_id}"
  path_part   = "mytestresource"
}

resource "aws_api_gateway_method" "test" {
  rest_api_id   = "${aws_api_gateway_rest_api.test.id}"
  resource_id   = "${aws_api_gateway_resource.test.id}"
  http_method   = "GET"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_resource.test.id}"
  http_method = "${aws_api_gateway_method.test.http_method}"
  type        = "MOCK"

  request_templates {
    "application/xml" = <<EOF
{
   "body" : $input.json('$')
}
EOF
  }
}
```

## Argument Reference

The following arguments are supported:

* `rest_api_id` - (Required) The ID of the REST API
* `stage_name` - (Required) The name of the stage
* `method_path` - (Required) Method path defined as `{resource_path}/{http_method}` for an individual method override, or `*/*` for overriding all methods in the stage.
* `settings` - (Required) The settings block, see below.

### `settings`

* `metrics_enabled` - (Optional) Specifies whether Amazon CloudWatch metrics are enabled for this method.
* `logging_level` - (Optional) Specifies the logging level for this method, which effects the log entries pushed to Amazon CloudWatch Logs. The available levels are `OFF`, `ERROR`, and `INFO`.
* `data_trace_enabled` - (Optional) Specifies whether data trace logging is enabled for this method, which effects the log entries pushed to Amazon CloudWatch Logs.
* `throttling_burst_limit` - (Optional) Specifies the throttling burst limit.
* `throttling_rate_limit` - (Optional) Specifies the throttling rate limit.
* `caching_enabled` - (Optional) Specifies whether responses should be cached and returned for requests. A cache cluster must be enabled on the stage for responses to be cached. 
* `cache_ttl_in_seconds` - (Optional) Specifies the time to live (TTL), in seconds, for cached responses. The higher the TTL, the longer the response will be cached.
* `cache_data_encrypted` - (Optional) Specifies whether the cached responses are encrypted.
* `require_authorization_for_cache_control` - (Optional) Specifies whether authorization is required for a cache invalidation request.
* `unauthorized_cache_control_header_strategy` - (Optional) Specifies how to handle unauthorized requests for cache invalidation. The available values are `FAIL_WITH_403`, `SUCCEED_WITH_RESPONSE_HEADER`, `SUCCEED_WITHOUT_RESPONSE_HEADER`.
