---
layout: "aws"
page_title: "AWS: cloudfront_distribution"
sidebar_current: "docs-aws-resource-cloudfront-distribution"
description: |-
  Provides a CloudFront web distribution resource.
---

# aws\_cloudfront\_distribution

Creates an Amazon CloudFront web distribution.

For information about CloudFront distributions, see the
[Amazon CloudFront Developer Guide][1]. For specific information about creating
CloudFront web distributions, see the [POST Distribution][2] page in the Amazon
CloudFront API Reference.

~> **NOTE:** CloudFront distributions take about 15 minutes to a deployed state
after creation or modification. During this time, deletes to resources will be
blocked. If you need to delete a distribution that is enabled and you do not
want to wait, you need to use the `retain_on_delete` flag.

## Example Usage

The following example below creates a CloudFront distribution with an S3 origin.

```hcl
resource "aws_s3_bucket" "b" {
  bucket = "mybucket"
  acl    = "private"

  tags {
    Name = "My bucket"
  }
}

resource "aws_cloudfront_distribution" "s3_distribution" {
  origin {
    domain_name = "${aws_s3_bucket.b.bucket_domain_name}"
    origin_id   = "myS3Origin"

    s3_origin_config {
      origin_access_identity = "origin-access-identity/cloudfront/ABCDEFG1234567"
    }
  }

  enabled             = true
  is_ipv6_enabled     = true
  comment             = "Some comment"
  default_root_object = "index.html"

  logging_config {
    include_cookies = false
    bucket          = "mylogs.s3.amazonaws.com"
    prefix          = "myprefix"
  }

  aliases = ["mysite.example.com", "yoursite.example.com"]

  default_cache_behavior {
    allowed_methods  = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
    cached_methods   = ["GET", "HEAD"]
    target_origin_id = "myS3Origin"

    forwarded_values {
      query_string = false

      cookies {
        forward = "none"
      }
    }

    viewer_protocol_policy = "allow-all"
    min_ttl                = 0
    default_ttl            = 3600
    max_ttl                = 86400
  }

  price_class = "PriceClass_200"

  restrictions {
    geo_restriction {
      restriction_type = "whitelist"
      locations        = ["US", "CA", "GB", "DE"]
    }
  }

  tags {
    Environment = "production"
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }
}
```

## Argument Reference

The CloudFront distribution argument layout is a complex structure composed
of several sub-resources - these resources are laid out below.

### Top-Level Arguments

  * `aliases` (Optional) - Extra CNAMEs (alternate domain names), if any, for
    this distribution.

  * `cache_behavior` (Optional) - A [cache behavior](#cache-behavior-arguments)
    resource for this distribution (multiples allowed).

  * `comment` (Optional) - Any comments you want to include about the
    distribution.

  * `custom_error_response` (Optional) - One or more [custom error response](#custom-error-response-arguments) elements (multiples allowed).

  * `default_cache_behavior` (Required) - The [default cache behavior](#default-cache-behavior-arguments) for this distribution (maximum
    one).

  * `default_root_object` (Optional) - The object that you want CloudFront to
    return (for example, index.html) when an end user requests the root URL.

  * `enabled` (Required) - Whether the distribution is enabled to accept end
    user requests for content.

  * `is_ipv6_enabled` (Optional) - Whether the IPv6 is enabled for the distribution.

  * `http_version` (Optional) - The maximum HTTP version to support on the
    distribution. Allowed values are `http1.1` and `http2`. The default is
    `http2`.

  * `logging_config` (Optional) - The [logging
    configuration](#logging-config-arguments) that controls how logs are written
    to your distribution (maximum one).

  * `origin` (Required) - One or more [origins](#origin-arguments) for this
    distribution (multiples allowed).

  * `price_class` (Optional) - The price class for this distribution. One of
    `PriceClass_All`, `PriceClass_200`, `PriceClass_100`

  * `restrictions` (Required) - The [restriction
    configuration](#restrictions-arguments) for this distribution (maximum one).

  * `tags` - (Optional) A mapping of tags to assign to the resource.

  * `viewer_certificate` (Required) - The [SSL
    configuration](#viewer-certificate-arguments) for this distribution (maximum
    one).

  * `web_acl_id` (Optional) - If you're using AWS WAF to filter CloudFront
    requests, the Id of the AWS WAF web ACL that is associated with the
    distribution.

  * `retain_on_delete` (Optional) - Disables the distribution instead of
    deleting it when destroying the resource through Terraform. If this is set,
    the distribution needs to be deleted manually afterwards. Default: `false`.

#### Cache Behavior Arguments

  * `allowed_methods` (Required) - Controls which HTTP methods CloudFront
    processes and forwards to your Amazon S3 bucket or your custom origin.

  * `cached_methods` (Required) - Controls whether CloudFront caches the
    response to requests using the specified HTTP methods.

  * `compress` (Optional) - Whether you want CloudFront to automatically
    compress content for web requests that include `Accept-Encoding: gzip` in
    the request header (default: `false`).

  * `default_ttl` (Required) - The default amount of time (in seconds) that an
    object is in a CloudFront cache before CloudFront forwards another request
    in the absence of an `Cache-Control max-age` or `Expires` header.

  * `forwarded_values` (Required) - The [forwarded values configuration](#forwarded-values-arguments) that specifies how CloudFront
    handles query strings, cookies and headers (maximum one).

  * `lambda_function_association` (Optional) - A config block that triggers a lambda function with
  specific actions. Defined below, maximum 4. **Lambda@Edge is in technical
  Preview, and must be enabled on your AWS account to be used**

  * `max_ttl` (Required) - The maximum amount of time (in seconds) that an
    object is in a CloudFront cache before CloudFront forwards another request
    to your origin to determine whether the object has been updated. Only
    effective in the presence of `Cache-Control max-age`, `Cache-Control
    s-maxage`, and `Expires` headers.

  * `min_ttl` (Required) - The minimum amount of time that you want objects to
    stay in CloudFront caches before CloudFront queries your origin to see
    whether the object has been updated.

  * `path_pattern` (Required) - The pattern (for example, `images/*.jpg)` that
    specifies which requests you want this cache behavior to apply to.

  * `smooth_streaming` (Optional) - Indicates whether you want to distribute
    media files in Microsoft Smooth Streaming format using the origin that is
    associated with this cache behavior.

  * `target_origin_id` (Required) - The value of ID for the origin that you want
    CloudFront to route requests to when a request matches the path pattern
    either for a cache behavior or for the default cache behavior.

  * `trusted_signers` (Optional) - The AWS accounts, if any, that you want to
    allow to create signed URLs for private content.

  * `viewer_protocol_policy` (Required) - Use this element to specify the
    protocol that users can use to access the files in the origin specified by
    TargetOriginId when a request matches the path pattern in PathPattern. One
    of `allow-all`, `https-only`, or `redirect-to-https`.

##### Forwarded Values Arguments

  * `cookies` (Required) - The [forwarded values cookies](#cookies-arguments)
    that specifies how CloudFront handles cookies (maximum one).

  * `headers` (Optional) - Specifies the Headers, if any, that you want
    CloudFront to vary upon for this cache behavior. Specify `*` to include all
    headers.

  * `query_string` (Required) - Indicates whether you want CloudFront to forward
    query strings to the origin that is associated with this cache behavior.

  * `query_string_cache_keys` (Optional) - When specified, along with a value of
    `true` for `query_string`, all query strings are forwarded, however only the
    query string keys listed in this argument are cached. When omitted with a
    value of `true` for `query_string`, all query string keys are cached.

##### Lambda Function Association

Lambda@Edge allows you to associate an AWS Lambda Function with a predefined
event. You can associate a single function per event type. See [What is
Lambda@Edge](http://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/what-is-lambda-at-edge.html)
for more information

  * `event_type` (Required) - The specific event to trigger this function.
  Valid values: `viewer-request`, `origin-request`, `viewer-response`,
  `origin-response`

  * `lambda_arn` (Required) - ARN of the Lambda function.

##### Cookies Arguments

  * `forward` (Required) - Specifies whether you want CloudFront to forward
    cookies to the origin that is associated with this cache behavior. You can
    specify `all`, `none` or `whitelist`. If `whitelist`, you must include the
    subsequent `whitelisted_names`

  * `whitelisted_names` (Optional) - If you have specified `whitelist` to
    `forward`, the whitelisted cookies that you want CloudFront to forward to
    your origin.

#### Custom Error Response Arguments

  * `error_caching_min_ttl` (Optional) - The minimum amount of time you want
    HTTP error codes to stay in CloudFront caches before CloudFront queries your
    origin to see whether the object has been updated.

  * `error_code` (Required) - The 4xx or 5xx HTTP status code that you want to
    customize.

  * `response_code` (Optional) - The HTTP status code that you want CloudFront
    to return with the custom error page to the viewer.

  * `response_page_path` (Optional) - The path of the custom error page (for
    example, `/custom_404.html`).

#### Default Cache Behavior Arguments

The arguments for `default_cache_behavior` are the same as for
[`cache_behavior`](#cache-behavior-arguments), except for the `path_pattern`
argument is not required.

#### Logging Config Arguments

  * `bucket` (Required) - The Amazon S3 bucket to store the access logs in, for
    example, `myawslogbucket.s3.amazonaws.com`.

  * `include_cookies` (Optional) - Specifies whether you want CloudFront to
    include cookies in access logs (default: `false`).

  * `prefix` (Optional) - An optional string that you want CloudFront to prefix
    to the access log filenames for this distribution, for example, `myprefix/`.

#### Origin Arguments

  * `custom_origin_config` - The [CloudFront custom
    origin](#custom-origin-config-arguments) configuration information. If an S3
    origin is required, use `s3_origin_config` instead.

  * `domain_name` (Required) - The DNS domain name of either the S3 bucket, or
    web site of your custom origin.

  * `custom_header` (Optional) - One or more sub-resources with `name` and
    `value` parameters that specify header data that will be sent to the origin
    (multiples allowed).

  * `origin_id` (Required) - A unique identifier for the origin.

  * `origin_path` (Optional) - An optional element that causes CloudFront to
    request your content from a directory in your Amazon S3 bucket or your
    custom origin.

  * `s3_origin_config` - The [CloudFront S3 origin](#s3-origin-config-arguments)
    configuration information. If a custom origin is required, use
    `custom_origin_config` instead.

##### Custom Origin Config Arguments

  * `http_port` (Required) - The HTTP port the custom origin listens on.

  * `https_port` (Required) - The HTTPS port the custom origin listens on.

  * `origin_protocol_policy` (Required) - The origin protocol policy to apply to
    your origin. One of `http-only`, `https-only`, or `match-viewer`.

  * `origin_ssl_protocols` (Required) - The SSL/TLS protocols that you want
    CloudFront to use when communicating with your origin over HTTPS. A list of
    one or more of `SSLv3`, `TLSv1`, `TLSv1.1`, and `TLSv1.2`.

  * `origin_keepalive_timeout` - (Optional) The Custom KeepAlive timeout, in seconds. Value must be between `1` and `60`.

  * `origin_read_timeout` - (Optional) The Custom Read timeout, in seconds. Value must be between `4` and `60`.

##### S3 Origin Config Arguments

* `origin_access_identity` (Optional) - The [CloudFront origin access
  identity][5] to associate with the origin.

#### Restrictions Arguments

The `restrictions` sub-resource takes another single sub-resource named
`geo_restriction` (see the example for usage).

The arguments of `geo_restriction` are:

  * `locations` (Optional) - The [ISO 3166-1-alpha-2 codes][4] for which you
    want CloudFront either to distribute your content (`whitelist`) or not
    distribute your content (`blacklist`).

  * `restriction_type` (Required) - The method that you want to use to restrict
    distribution of your content by country: `none`, `whitelist`, or
    `blacklist`.

#### Viewer Certificate Arguments

  * `acm_certificate_arn` - The ARN of the [AWS Certificate Manager][6]
    certificate that you wish to use with this distribution. Specify this,
    `cloudfront_default_certificate`, or `iam_certificate_id`.  The ACM
    certificate must be in  US-EAST-1.

  * `cloudfront_default_certificate` - `true` if you want viewers to use HTTPS
    to request your objects and you're using the CloudFront domain name for your
    distribution. Specify this, `acm_certificate_arn`, or `iam_certificate_id`.

  * `iam_certificate_id` - The IAM certificate identifier of the custom viewer
    certificate for this distribution if you are using a custom domain. Specify
    this, `acm_certificate_arn`, or `cloudfront_default_certificate`.

  * `minimum_protocol_version` - The minimum version of the SSL protocol that
    you want CloudFront to use for HTTPS connections. One of `SSLv3` or `TLSv1`.
    Default: `SSLv3`. **NOTE**: If you are using a custom certificate (specified
    with `acm_certificate_arn` or `iam_certificate_id`), and have specified
    `sni-only` in `ssl_support_method`, `TLSv1` must be specified.

  * `ssl_support_method`: Specifies how you want CloudFront to serve HTTPS
    requests. One of `vip` or `sni-only`. Required if you specify
    `acm_certificate_arn` or `iam_certificate_id`. **NOTE:** `vip` causes
    CloudFront to use a dedicated IP address and may incur extra charges.

## Attribute Reference

The following attributes are exported:

  * `id` - The identifier for the distribution. For example: `EDFDVBD632BHDS5`.

  * `arn` - The ARN (Amazon Resource Name) for the distribution. For example: arn:aws:cloudfront::123456789012:distribution/EDFDVBD632BHDS5, where 123456789012 is your AWS account ID.

  * `caller_reference` - Internal value used by CloudFront to allow future
    updates to the distribution configuration.

  * `status` - The current status of the distribution. `Deployed` if the
    distribution's information is fully propagated throughout the Amazon
    CloudFront system.

  * `active_trusted_signers` - The key pair IDs that CloudFront is aware of for
    each trusted signer, if the distribution is set up to serve private content
    with signed URLs.

  * `domain_name` - The domain name corresponding to the distribution. For
    example: `d604721fxaaqy9.cloudfront.net`.

  * `last_modified_time` - The date and time the distribution was last modified.

  * `in_progress_validation_batches` - The number of invalidation batches
    currently in progress.

  * `etag` - The current version of the distribution's information. For example:
    `E2QWRUHAPOMQZL`.

  * `hosted_zone_id` - The CloudFront Route 53 zone ID that can be used to
     route an [Alias Resource Record Set][7] to. This attribute is simply an
     alias for the zone ID `Z2FDTNDATAQYW2`.


[1]: http://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/Introduction.html
[2]: http://docs.aws.amazon.com/AmazonCloudFront/latest/APIReference/CreateDistribution.html
[3]: http://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/private-content-restricting-access-to-s3.html
[4]: http://www.iso.org/iso/country_codes/iso_3166_code_lists/country_names_and_code_elements.htm
[5]: /docs/providers/aws/r/cloudfront_origin_access_identity.html
[6]: https://aws.amazon.com/certificate-manager/
[7]: http://docs.aws.amazon.com/Route53/latest/APIReference/CreateAliasRRSAPI.html


## Import

Cloudfront Distributions can be imported using the `id`, e.g.

```
$ terraform import aws_cloudfront_distribution.distribution E74FTE3EXAMPLE
```
