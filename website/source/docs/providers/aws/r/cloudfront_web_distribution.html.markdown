---
layout: "aws"
page_title: "AWS: cloudfront_web_distribution"
sidebar_current: "docs-aws-resource-cloudfront-web-distribution"
description: |-
  Provides a CloudFront web distribution resource.
---

# cloudfront\_web\_distribution

Provides a CloudFront distribution resource. Distributions takes approximately
15 minutes to deploy.

## Example Usage

```
resource "aws_cloudfront_web_distribution" "static" {
  origin_domain_name = "bucket.s3.amazonaws.com"
}
```

## Argument Reference

The following arguments are supported:

* `origin_domain_name` - (Required) The Amazon S3 bucket or web server from which you want CloudFront to get your web content.
* `enabled` - (Optional) Default: `true`.
* `comment` - (Optional)
* `price_class` - (Optional) Default: `"PriceClass_All"`.
* `default_root_object` - (Optional)
* `origin_path` - (Optional) Request the content from a directory in your Amazon S3 bucket or your custom origin.
* `http_port` - (Optional) Default: `80`.
* `https_port` - (Optional) Default: `443`.
* `origin_protocol_policy` - (Optional) Default: `"http-only"`.
* `viewer_protocol_policy` - (Optional) Default: `"allow-all"`.
* `forward_cookie` - (Optional) Include all user cookies in the request URLs that it forwards to your origin. Default: `"none"`.
* `forward_query_string` - (Optional) Include query strings in the request URLs that it forwards to your origin. Default: `false`.
* `logging_enabled` - (Optional) Log all viewer requests for files in your distribution. Default: `false`.
* `logging_include_cookies` - (Optional) Include cookies in access logs. Default: `false`.
* `logging_prefix` - (Optional) Prefix for the names of log files.
* `logging_bucket` - (Optional) Destination bucket in the format of `bucketname.s3.amazonaws.com`.
* `minimum_ttl` - (Optional) The minimum amount of time (in seconds) that an object is in a CloudFront cache before CloudFront forwards another request to your origin to determine whether an updated version is available.
* `aliases` - (Optional) A list alternate domain names(CNAMES).
* `geo_restriction_type` - (Optional) Type of restriction. Default `"none"`.
* `geo_restrictions` - (Optional) A list of two-letter country codes.

## Attributes Reference

The following attributes are exported:

* `id` - The unique identifier of the distribution.
* `domain_name` - Unique domain of the resource.
* `zone_id` - The canonical hosted zone ID of CloudFront(to be used in a Route 53 Alias record)
