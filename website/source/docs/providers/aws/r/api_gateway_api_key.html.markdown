---
layout: "aws"
page_title: "AWS: aws_api_gateway_api_key"
sidebar_current: "docs-aws-resource-api-gateway-api-key"
description: |-
  Provides an API Gateway API Key.
---

# aws\_api\_gateway\_api\_key

Provides an API Gateway API Key.

## Example Usage

```
resource "aws_api_gateway_rest_api" "MyDemoAPI" {
  name = "MyDemoAPI"
  description = "This is my API for demonstration purposes"
}

resource "aws_api_gateway_api_key" "MyDemoApiKey" {
  name = "demo"

  stage_key {
    rest_api_id = "${aws_api_gateway_rest_api.MyDemoAPI.id}"
    stage_name = "${aws_api_gateway_deployment.MyDemoDeployment.stage_name}"
  }
}

resource "aws_api_gateway_deployment" "MyDemoDeployment" {
  rest_api_id = "${aws_api_gateway_rest_api.MyDemoAPI.id}"
  stage_name = "test"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the API key
* `description` - (Required) The API key description
* `enabled` - (Optional) Specifies whether the API key can be used by callers. Defaults to `true`.
* `stage_key` - (Optional) A list of stage keys associated with the API key - see below

`stage_key` block supports the following:

* `rest_api_id` - (Required) The ID of the associated REST API.
* `stage_name` - (Required) The name of the API Gateway stage.

## Attribute Reference

The following attributes are exported:

* `id` - The ID of the API key
