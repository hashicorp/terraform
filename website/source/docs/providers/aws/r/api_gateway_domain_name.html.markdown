---
layout: "aws"
page_title: "AWS: aws_api_gateway_domain_name"
sidebar_current: "docs-aws-resource-api-gateway-domain-name"
description: |-
  Provides an API Gateway Domain Name.
---

# aws\_api\_gateway\_deployment

Provides an API Gateway Domain Name.

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

resource "aws_api_gateway_domain" "MyDemoDomain" {
  domain_name = "your.doma.in"
  certificate_name = "demo_api_cert"
  certificate_body = "${file("certs/your.doma.in.crt")}"
  certificate_private_key = "${file("certs/your.doma.in.pkey")}"
  certificate_chain = "${file("certs/your.doma.in.chain")}"
}

resource "aws_api_gateway_base_path_mapping" "test" {
  api_id = "${aws_api_gateway_rest_api.MyDemoAPI.id}"
  path = ""
  stage = "${aws_api_gateway_deployment.MyDemoDeployment.stage_name}"
  domain_name = "${aws_api_gateway_domain.MyDemoDomain.id}"
}

```

## Argument Reference

The following arguments are supported:

* `domain_name` - (Required) The name of the domain name resource
* `certificate_name` - (Required) The name of the certificate.
* `certificate_body` - (Required) The body of the server certificate provided by your certificate authority.
* `certificate_private_key` - (Required) Your certificate's private key.
* `certificate_chain` - (Required) The intermediate certificates and optionally the root certificate, one after the other without any blank lines. If you include the root certificate, your certificate chain must start with intermediate certificates and end with the root certificate. Use the intermediate certificates that were provided by your certificate authority. Do not include any intermediaries that are not in the chain of trust path.


## Attribute Reference

The following attributes are exported:

* `distribution_domain` - The CloudFront distribution domain  
