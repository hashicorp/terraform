---
layout: "aws"
page_title: "AWS: cloudtrail"
sidebar_current: "docs-aws-resource-cloudtrail"
description: |-
  Provides a CloudTrail resource.
---

# aws\_cloudtrail

Provides a CloudTrail resource.

## Example Usage
```
resource "aws_cloudtrail" "foobar" {
    name = "tf-trail-foobar"
    s3_bucket_name = "${aws_s3_bucket.foo.id}"
    s3_key_prefix = "/prefix"
    include_global_service_events = false
}

resource "aws_s3_bucket" "foo" {
    bucket = "tf-test-trail"
    force_destroy = true
    policy = <<POLICY
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "AWSCloudTrailAclCheck",
            "Effect": "Allow",
            "Principal": {
              "Service": "cloudtrail.amazonaws.com"
            },
            "Action": "s3:GetBucketAcl",
            "Resource": "arn:aws:s3:::tf-test-trail"
        },
        {
            "Sid": "AWSCloudTrailWrite",
            "Effect": "Allow",
            "Principal": {
              "Service": "cloudtrail.amazonaws.com"
            },
            "Action": "s3:PutObject",
            "Resource": "arn:aws:s3:::tf-test-trail/*",
            "Condition": {
                "StringEquals": {
                    "s3:x-amz-acl": "bucket-owner-full-control"
                }
            }
        }
    ]
}
POLICY
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the trail.
* `s3_bucket_name` - (Required) Specifies the name of the S3 bucket designated for publishing log files.
* `s3_key_prefix` - (Optional) Specifies the S3 key prefix that precedes
    the name of the bucket you have designated for log file delivery.
* `cloud_watch_logs_role_arn` - (Optional) Specifies the role for the CloudWatch Logs
    endpoint to assume to write to a user’s log group.
* `cloud_watch_logs_group_arn` - (Optional) Specifies a log group name using an Amazon Resource Name (ARN),
    that represents the log group to which CloudTrail logs will be delivered.
* `enable_logging` - (Optional) Enables logging for the trail. Defaults to `true`.
    Setting this to `false` will pause logging.
* `include_global_service_events` - (Optional) Specifies whether the trail is publishing events
    from global services such as IAM to the log files. Defaults to `true`.
* `is_multi_region_trail` - (Optional) Specifies whether the trail is created in the current
    region or in all regions. Defaults to `false`.
* `sns_topic_name` - (Optional) Specifies the name of the Amazon SNS topic
    defined for notification of log file delivery.
* `enable_log_file_validation` - (Optional) Specifies whether log file integrity validation is enabled.
    Defaults to `false`.
* `kms_key_id` - (Optional) Specifies the KMS key ID to use to encrypt the logs delivered by CloudTrail.
* `tags` - (Optional) A mapping of tags to assign to the trail

## Attribute Reference

The following attributes are exported:

* `id` - The name of the trail.
* `home_region` - The region in which the trail was created.
* `arn` - The Amazon Resource Name of the trail.
