---
layout: "aws"
page_title: "AWS: aws_elastictranscoder_pipeline"
sidebar_current: "docs-aws-resource-elastic-transcoder-pipeline"
description: |-
  Provides an Elastic Transcoder pipeline resource.
---

# aws\_elastictranscoder\_pipeline

Provides an Elastic Transcoder pipeline resource.

## Example Usage

```hcl
resource "aws_elastictranscoder_pipeline" "bar" {
  input_bucket = "${aws_s3_bucket.input_bucket.bucket}"
  name         = "aws_elastictranscoder_pipeline_tf_test_"
  role         = "${aws_iam_role.test_role.arn}"

  content_config = {
    bucket        = "${aws_s3_bucket.content_bucket.bucket}"
    storage_class = "Standard"
  }

  thumbnail_config = {
    bucket        = "${aws_s3_bucket.thumb_bucket.bucket}"
    storage_class = "Standard"
  }
}
```

## Argument Reference

See ["Create Pipeline"](http://docs.aws.amazon.com/elastictranscoder/latest/developerguide/create-pipeline.html) in the AWS docs for reference.

The following arguments are supported:

* `aws_kms_key_arn` - (Optional) The AWS Key Management Service (AWS KMS) key that you want to use with this pipeline.
* `content_config` - (Optional) The ContentConfig object specifies information about the Amazon S3 bucket in which you want Elastic Transcoder to save transcoded files and playlists. (documented below)
* `content_config_permissions` - (Optional) The permissions for the `content_config` object. (documented below)
* `input_bucket` - (Required) The Amazon S3 bucket in which you saved the media files that you want to transcode and the graphics that you want to use as watermarks.
* `name` - (Optional, Forces new resource) The name of the pipeline. Maximum 40 characters
* `notifications` - (Optional) The Amazon Simple Notification Service (Amazon SNS) topic that you want to notify to report job status. (documented below)
* `output_bucket` - (Optional) The Amazon S3 bucket in which you want Elastic Transcoder to save the transcoded files.
* `role` - (Required) The IAM Amazon Resource Name (ARN) for the role that you want Elastic Transcoder to use to transcode jobs for this pipeline.
* `thumbnail_config` - (Optional) The ThumbnailConfig object specifies information about the Amazon S3 bucket in which you want Elastic Transcoder to save thumbnail files. (documented below)
* `thumbnail_config_permissions` - (Optional) The permissions for the `thumbnail_config` object. (documented below)

The `content_config` object specifies information about the Amazon S3 bucket in
which you want Elastic Transcoder to save transcoded files and playlists: which
bucket to use, and the storage class that you want to assign to the files. If
you specify values for `content_config`, you must also specify values for
`thumbnail_config`. If you specify values for `content_config` and
`thumbnail_config`, omit the `output_bucket` object.

The `content_config` object supports the following:

* `bucket` - The Amazon S3 bucket in which you want Elastic Transcoder to save transcoded files and playlists.
* `storage_class` - The Amazon S3 storage class, Standard or ReducedRedundancy, that you want Elastic Transcoder to assign to the files and playlists that it stores in your Amazon S3 bucket.

The `content_config_permissions` object supports the following:

* `access` - The permission that you want to give to the AWS user that you specified in `content_config_permissions.grantee`
* `grantee` - The AWS user or group that you want to have access to transcoded files and playlists.
* `grantee_type` - Specify the type of value that appears in the `content_config_permissions.grantee` object. Valid values are `Canonical`, `Email` or `Group`.


The `notifications` object supports the following:

* `completed` - The topic ARN for the Amazon SNS topic that you want to notify when Elastic Transcoder has finished processing a job in this pipeline.
* `error` - The topic ARN for the Amazon SNS topic that you want to notify when Elastic Transcoder encounters an error condition while processing a job in this pipeline.
* `progressing` - The topic ARN for the Amazon Simple Notification Service (Amazon SNS) topic that you want to notify when Elastic Transcoder has started to process a job in this pipeline.
* `warning` - The topic ARN for the Amazon SNS topic that you want to notify when Elastic Transcoder encounters a warning condition while processing a job in this pipeline.

The `thumbnail_config` object specifies information about the Amazon S3 bucket in
which you want Elastic Transcoder to save thumbnail files: which bucket to use,
which users you want to have access to the files, the type of access you want
users to have, and the storage class that you want to assign to the files. If
you specify values for `content_config`, you must also specify values for
`thumbnail_config` even if you don't want to create thumbnails. (You control
whether to create thumbnails when you create a job. For more information, see
ThumbnailPattern in the topic Create Job.) If you specify values for
`content_config` and `thumbnail_config`, omit the OutputBucket object.

The `thumbnail_config` object supports the following:

* `bucket` - The Amazon S3 bucket in which you want Elastic Transcoder to save thumbnail files.
* `storage_class` - The Amazon S3 storage class, Standard or ReducedRedundancy, that you want Elastic Transcoder to assign to the thumbnails that it stores in your Amazon S3 bucket.

The `thumbnail_config_permissions` object supports the following:

* `access` - The permission that you want to give to the AWS user that you specified in `thumbnail_config_permissions.grantee`.
* `grantee` - The AWS user or group that you want to have access to thumbnail files.
* `grantee_type` - Specify the type of value that appears in the `thumbnail_config_permissions.grantee` object.
