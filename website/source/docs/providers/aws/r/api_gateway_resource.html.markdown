---
layout: "aws"
page_title: "AWS: aws_api_gateway_resource"
sidebar_current: "docs-aws-resource-api-gateway-resource"
description: |-
  Provides an API Gateway Resource.
---

# aws\_api\_gateway\_resource

Provides an API Gateway REST API Resource.

## Example Usage

```
resource "aws_api_gateway_rest_api" "MyDemoAPI" {
  name = "MyDemoAPI"
  description = "This is my API for demonstration purposes"
}

resource "aws_api_gateway_resource" "MyDemoResource" {
  rest_api_id = "${aws_api_gateway_rest_api.MyDemoAPI.id}"
  parent_id = "${aws_api_gateway_rest_api.MyDemoAPI.root_resource_id}"
  path_part = "mydemoresource"
}
```

## Argument Reference

The following arguments are supported:

* `rest_api_id` - (Required) API Gateway ID
* `parent_id` - (Required) Parent resource ID
* `path_part` - (Required) The resource path

## Attributes Reference

The following attributes are exported:

* `path` - The complete path for this resource, including all parent paths
