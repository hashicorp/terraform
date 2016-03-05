---
layout: "aws"
page_title: "AWS: aws_api_gateway_model"
sidebar_current: "docs-aws-resource-api-gateway-model"
description: |-
  Provides a Model for a API Gateway.
---

# aws\_api\_gateway\_model

Provides a Model for a API Gateway.

## Example Usage

```
resource "aws_api_gateway_rest_api" "MyDemoAPI" {
  name = "MyDemoAPI"
  description = "This is my API for demonstration purposes"
}

resource "aws_api_gateway_model" "MyDemoModel" {
  rest_api_id = "${aws_api_gateway_rest_api.MyDemoAPI.id}"
  name = "user"
  description = "a JSON schema"
  content_type = "application/json"
  schema = <<EOF
{
  "type": "object"
}
EOF
}
```

## Argument Reference

The following arguments are supported:

* `rest_api_id` - (Required) API Gateway ID
* `name` - (Required) Name of the model
* `description` - (Optional) Model description
* `content_type` - (Required) Model content type
* `schema` - (Required) Model schema
