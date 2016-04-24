---
layout: "aws"
page_title: "AWS: aws_api_gateway_swagger_api"
sidebar_current: "docs-aws-resource-api-gateway-swagger-api"
description: |-
  Provides an API Gateway REST API via a Swagger import
---

# aws\_api\_gateway\_swagger\_api

Provides an API Gateway REST API via a Swagger import.

## Example Usage

```
resource "aws_api_gateway_swagger_api" "MyDemoAPI" {
  swagger = <<EOF
{
  "swagger": "2.0",
  "info": {
    "version": "1.0",
    "title": "Hello World API"
  },
...
}
EOF         
}
```

## Argument Reference

The following arguments are supported:

* `swagger` - (Required) The swagger defintion, in JSON format
* `failonwarnings` - (Optional) Tells AWS to promote swagger warnings to errors

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the REST API
