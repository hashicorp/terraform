---
layout: "aws"
page_title: "AWS: aws_api_gateway_deployment"
sidebar_current: "docs-aws-resource-api-gateway-deployment"
description: |-
  Provides an API Gateway Deployment.
---

# aws\_api\_gateway\_deployment

Provides an API Gateway Deployment.

-> **Note:** Depends on having `aws_api_gateway_method` inside your rest api. To ensure this
you might need to add an explicit `depends_on` for clean runs.

## Example Usage

```
resource "aws_api_gateway_rest_api" "MyDemoAPI" {
  name = "MyDemoAPI"
  description = "This is my API for demonstration purposes"
}

resource "aws_api_gateway_resource" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.MyDemoAPI.id}"
  parent_id = "${aws_api_gateway_rest_api.MyDemoAPI.root_resource_id}"
  path_part = "test"
}

resource "aws_api_gateway_method" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.MyDemoAPI.id}"
  resource_id = "${aws_api_gateway_resource.test.id}"
  http_method = "GET"
  authorization = "NONE"
}

resource "aws_api_gateway_deployment" "MyDemoDeployment" {
  depends_on = ["aws_api_gateway_integration.test"]

  rest_api_id = "${aws_api_gateway_rest_api.MyDemoAPI.id}"

  variables = {
    "answer" = "42"
  }
}
```

## Argument Reference

The following arguments are supported:

* `rest_api_id` - (Required) The ID of the associated REST API
* `stage_name` - (Optional) The name of the stage
* `description` - (Optional) The description of the deployment
* `stage_description` - (Optional) The description of the stage
* `variables` - (Optional) A map that defines variables for the stage

## Attribute Reference

The following attributes are exported:

* `id` - The ID of the deployment
