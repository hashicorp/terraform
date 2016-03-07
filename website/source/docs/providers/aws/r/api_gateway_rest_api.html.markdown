---
layout: "aws"
page_title: "AWS: aws_api_gateway_rest_api"
sidebar_current: "docs-aws-resource-api-gateway-rest-api"
description: |-
  Provides an API Gateway REST API.
---

# aws\_api\_gateway\_rest\_api

Provides an API Gateway REST API.

## Example Usage

```
resource "aws_api_gateway_rest_api" "MyDemoAPI" {
  name = "MyDemoAPI"
  description = "This is my API for demonstration purposes"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Name of the API Gateway
* `description` - (Optional) The API Gateway description

## Attributes Reference

The following attributes are exported:

* `root_resource_id` - The resource ID of the APIs root
