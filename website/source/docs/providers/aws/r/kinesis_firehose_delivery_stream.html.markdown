---
layout: "aws"
page_title: "AWS: aws_kinesis_firehose_delivery_stream"
sidebar_current: "docs-aws-resource-kinesis-firehose-delivery-stream"
description: |-
  Provides a AWS Kinesis Firehose Delivery Stream
---

# aws\_kinesis\_stream

Provides a Kinesis Firehose Delivery Stream resource. Amazon Kinesis Firehose is a fully managed, elastic service to easily deliver real-time data streams to destinations such as Amazon S3 and Amazon Redshift.

For more details, see the [Amazon Kinesis Firehose Documentation][1].

## Example Usage

```
resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket"
	acl = "private"
}

esource "aws_iam_role" "firehose_role" {
   name = "firehose_test_role"
   assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "firehose.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_kinesis_firehose_delivery_stream" "test_stream" {
	name = "terraform-kinesis-firehose-test-stream"
	destination = "s3"
	role_arn = "${aws_iam_role.firehose_role.arn}"
	s3_bucket_arn = "${aws_s3_bucket.bucket.arn}"
}
```

~> **NOTE:** Kinesis Firehose is currently only supported in us-east-1, us-west-2 and eu-west-1. This implementation of Kinesis Firehose only supports the s3 destination type as Terraform doesn't support Redshift yet.

## Argument Reference

The following arguments are supported:

* `name` - (Required) A name to identify the stream. This is unique to the 
AWS account and region the Stream is created in.
* `destination` – (Required) This is the destination to where the data is delivered. The only options are `s3` & `redshift`
* `role_arn` - (Required) The ARN of the AWS credentials.
* `s3_bucket_arn` - (Required) The ARN of the S3 bucket
* `s3_prefix` - (Optional) The "YYYY/MM/DD/HH" time format prefix is automatically used for delivered S3 files. You can specify an extra prefix to be added in front of the time format prefix. Note that if the prefix ends with a slash, it appears as a folder in the S3 bucket
* `s3_buffer_size` - (Optional) Buffer incoming data to the specified size, in MBs, before delivering it to the destination. The default value is 5.
                                We recommend setting SizeInMBs to a value greater than the amount of data you typically ingest into the delivery stream in 10 seconds. For example, if you typically ingest data at 1 MB/sec set SizeInMBs to be 10 MB or highe
* `s3_buffer_interval` - (Optional) Buffer incoming data for the specified period of time, in seconds, before delivering it to the destination. The default value is 300
* `s3_data_compression` - (Optional) The compression format. If no value is specified, the default is NOCOMPRESSION. Other supported values are GZIP, ZIP & Snappy 


## Attributes Reference

* `arn` - The Amazon Resource Name (ARN) specifying the Stream

[1]: http://aws.amazon.com/documentation/firehose/
