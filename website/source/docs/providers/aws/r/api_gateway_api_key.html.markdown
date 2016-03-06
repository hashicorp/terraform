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

* `name` - (Required) Name of the API Gateway
* `description` - (Optional) The API Gateway description
* `stage_key` - (Optional) applicable API Gateway stages

Stage keys support the following:

* `rest_api_id` - (Required) The ID of the associated API Gateway Rest API.
* `stage_name` - (Required) The name of the API Gateway stage.
