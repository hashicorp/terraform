---
layout: "aws"
page_title: "AWS: aws_route53_health_check"
sidebar_current: "docs-aws-resource-route53-health-check"
description: |-
  Provides a Route53 health check.
---
# aws\_route53\_health\_check

Provides a Route53 health check.

## Example Usage

```
resource "aws_route53_health_check" "child1" {
  fqdn = "foobar.terraform.com"
  port = 80
  type = "HTTP"
  resource_path = "/"
  failure_threshold = "5"
  request_interval = "30"

  tags = {
    Name = "tf-test-health-check"
   }
}

resource "aws_route53_health_check" "foo" {
  type = "CALCULATED"
  child_health_threshold = 1
  child_healthchecks = ["${aws_route53_health_check.child1.id}"]

  tags = {
    Name = "tf-test-calculated-health-check"
   }
}
```

## Argument Reference

The following arguments are supported:

* `fqdn` - (Optional) The fully qualified domain name of the endpoint to be checked.
* `ip_address` - (Optional) The IP address of the endpoint to be checked.
* `port` - (Optional) The port of the endpoint to be checked.
* `type` - (Required) The protocol to use when performing health checks. Valid values are `HTTP`, `HTTPS`, `HTTP_STR_MATCH`, `HTTPS_STR_MATCH`, `TCP` and `CALCULATED`.
* `failure_threshold` - (Required) The number of consecutive health checks that an endpoint must pass or fail.
* `request_interval` - (Required) The number of seconds between the time that Amazon Route 53 gets a response from your endpoint and the time that it sends the next health-check request.
* `resource_path` - (Optional) The path that you want Amazon Route 53 to request when performing health checks.
* `search_string` - (Optional) String searched in the first 5120 bytes of the response body for check to be considered healthy.
* `measure_latency` - (Optional) A Boolean value that indicates whether you want Route 53 to measure the latency between health checkers in multiple AWS regions and your endpoint and to display CloudWatch latency graphs in the Route 53 console.
* `invert_healthcheck` - (Optional) A boolean value that indicates whether the status of health check should be inverted. For example, if a health check is healthy but Inverted is True , then Route 53 considers the health check to be unhealthy.
* `child_healthchecks` - (Optional) For a specified parent health check, a list of HealthCheckId values for the associated child health checks.
* `child_health_threshold` - (Optional) The minimum number of child health checks that must be healthy for Route 53 to consider the parent health check to be healthy. Valid values are integers between 0 and 256, inclusive
* `tags` - (Optional) A mapping of tags to assign to the health check.

At least one of either `fqdn` or `ip_address` must be specified.

