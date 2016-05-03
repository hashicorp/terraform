---
layout: "aws"
page_title: "AWS: cloudfront_origin_access_identity"
sidebar_current: "docs-aws-resource-cloudfront-origin-access-identity"
description: |-
  Provides a CloudFront origin access identity.
---

# aws\_cloudfront\_origin\_access\_identity

Creates an Amazon CloudFront origin access identity.

For information about CloudFront distributions, see the
[Amazon CloudFront Developer Guide][1]. For more information on generating
origin access identities, see
[Using an Origin Access Identity to Restrict Access to Your Amazon S3 Content][2].

## Example Usage

The following example below creates a CloudFront origin access identity.

```
resource "aws_cloudfront_origin_access_identity" "origin_access_identity" {
  comment = "Some comment"
}
```

## Argument Reference

* `comment` (Optional) - An optional comment for the origin access identity.

## Attribute Reference

The following attributes are exported:

* `id` - The identifier for the distribution. For example: `EDFDVBD632BHDS5`.
* `caller_reference` - Internal value used by CloudFront to allow future updates to the origin access identity.
* `cloudfront_access_identity_path` - A shortcut to the full path for the origin access identity to use in CloudFront, see below.
* `etag` - The current version of the origin access identity's information. For example: E2QWRUHAPOMQZL.
* `s3_canonical_user_id` - The Amazon S3 canonical user ID for the origin access identity, which you use when giving the origin access identity read permission to an object in Amazon S3.

## Using With CloudFront

Normally, when referencing an origin access identity in CloudFront, you need to
prefix the ID with the `origin-access-identity/cloudfront/` special path.
The `cloudfront_access_identity_path` allows this to be circumvented.
The below snippet demonstrates use with the `s3_origin_config` structure for the
[`aws_cloudfront_web_distribution`][3] resource:

```
s3_origin_config {
  origin_access_identity = "${aws_cloudfront_origin_access_identity.origin_access_identity.cloudfront_access_identity_path}"
}
```

[1]: http://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/Introduction.html
[2]: http://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/private-content-restricting-access-to-s3.html
[3]: /docs/providers/aws/r/cloudfront_distribution.html
