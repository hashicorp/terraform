---
layout: "aws"
page_title: "AWS: aws_kinesis_firehose_delivery_stream"
sidebar_current: "docs-aws-resource-kinesis-firehose-delivery-stream"
description: |-
  Provides a AWS Kinesis Firehose Delivery Stream
---

# aws\_kinesis\_firehose\_delivery\_stream

Provides a Kinesis Firehose Delivery Stream resource. Amazon Kinesis Firehose is a fully managed, elastic service to easily deliver real-time data streams to destinations such as Amazon S3 and Amazon Redshift.

For more details, see the [Amazon Kinesis Firehose Documentation][1].

## Example Usage

### S3 Destination

```hcl
resource "aws_s3_bucket" "bucket" {
  bucket = "tf-test-bucket"
  acl    = "private"
}

resource "aws_iam_role" "firehose_role" {
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
  name        = "terraform-kinesis-firehose-test-stream"
  destination = "s3"

  s3_configuration {
    role_arn   = "${aws_iam_role.firehose_role.arn}"
    bucket_arn = "${aws_s3_bucket.bucket.arn}"
  }
}
```

### Redshift Destination

```hcl
resource "aws_redshift_cluster" "test_cluster" {
  cluster_identifier = "tf-redshift-cluster-%d"
  database_name      = "test"
  master_username    = "testuser"
  master_password    = "T3stPass"
  node_type          = "dc1.large"
  cluster_type       = "single-node"
}

resource "aws_kinesis_firehose_delivery_stream" "test_stream" {
  name        = "terraform-kinesis-firehose-test-stream"
  destination = "redshift"

  s3_configuration {
    role_arn           = "${aws_iam_role.firehose_role.arn}"
    bucket_arn         = "${aws_s3_bucket.bucket.arn}"
    buffer_size        = 10
    buffer_interval    = 400
    compression_format = "GZIP"
  }

  redshift_configuration {
    role_arn           = "${aws_iam_role.firehose_role.arn}"
    cluster_jdbcurl    = "jdbc:redshift://${aws_redshift_cluster.test_cluster.endpoint}/${aws_redshift_cluster.test_cluster.database_name}"
    username           = "testuser"
    password           = "T3stPass"
    data_table_name    = "test-table"
    copy_options       = "delimiter '|'" # the default delimiter
    data_table_columns = "test-col"
  }
}
```

### Elasticsearch Destination

```hcl
resource "aws_elasticsearch_domain" "test_cluster" {
  domain_name = "firehose-es-test"
}

resource "aws_kinesis_firehose_delivery_stream" "test_stream" {
  name        = "terraform-kinesis-firehose-test-stream"
  destination = "elasticsearch"

  s3_configuration {
    role_arn           = "${aws_iam_role.firehose_role.arn}"
    bucket_arn         = "${aws_s3_bucket.bucket.arn}"
    buffer_size        = 10
    buffer_interval    = 400
    compression_format = "GZIP"
  }

  elasticsearch_configuration {
    domain_arn = "${aws_elasticsearch_domain.test_cluster.arn}"
    role_arn   = "${aws_iam_role.firehose_role.arn}"
    index_name = "test"
    type_name  = "test"
  }
}
```

~> **NOTE:** Kinesis Firehose is currently only supported in us-east-1, us-west-2 and eu-west-1.

## Argument Reference

The following arguments are supported:

* `name` - (Required) A name to identify the stream. This is unique to the
AWS account and region the Stream is created in.
* `destination` – (Required) This is the destination to where the data is delivered. The only options are `s3`, `redshift`, and `elasticsearch`.
* `s3_configuration` - (Required) Configuration options for the s3 destination (or the intermediate bucket if the destination
is redshift). More details are given below.
* `redshift_configuration` - (Optional) Configuration options if redshift is the destination.
Using `redshift_configuration` requires the user to also specify a
`s3_configuration` block. More details are given below.

The `s3_configuration` object supports the following:

* `role_arn` - (Required) The ARN of the AWS credentials.
* `bucket_arn` - (Required) The ARN of the S3 bucket
* `prefix` - (Optional) The "YYYY/MM/DD/HH" time format prefix is automatically used for delivered S3 files. You can specify an extra prefix to be added in front of the time format prefix. Note that if the prefix ends with a slash, it appears as a folder in the S3 bucket
* `buffer_size` - (Optional) Buffer incoming data to the specified size, in MBs, before delivering it to the destination. The default value is 5.
                                We recommend setting SizeInMBs to a value greater than the amount of data you typically ingest into the delivery stream in 10 seconds. For example, if you typically ingest data at 1 MB/sec set SizeInMBs to be 10 MB or higher.
* `buffer_interval` - (Optional) Buffer incoming data for the specified period of time, in seconds, before delivering it to the destination. The default value is 300.
* `compression_format` - (Optional) The compression format. If no value is specified, the default is UNCOMPRESSED. Other supported values are GZIP, ZIP & Snappy. If the destination is redshift you cannot use ZIP or Snappy.
* `kms_key_arn` - (Optional) Specifies the KMS key ARN the stream will use to encrypt data. If not set, no encryption will
be used.
* `cloudwatch_logging_options` - (Optional) The CloudWatch Logging Options for the delivery stream. More details are given below

The `redshift_configuration` object supports the following:

* `cluster_jdbcurl` - (Required) The jdbcurl of the redshift cluster.
* `username` - (Required) The username that the firehose delivery stream will assume. It is strongly recommended that the username and password provided is used exclusively for Amazon Kinesis Firehose purposes, and that the permissions for the account are restricted for Amazon Redshift INSERT permissions.
* `password` - (Required) The password for the username above.
* `retry_duration` - (Optional) The length of time during which Firehose retries delivery after a failure, starting from the initial request and including the first attempt. The default value is 3600 seconds (60 minutes). Firehose does not retry if the value of DurationInSeconds is 0 (zero) or if the first delivery attempt takes longer than the current value.
* `role_arn` - (Required) The arn of the role the stream assumes.
* `data_table_name` - (Required) The name of the table in the redshift cluster that the s3 bucket will copy to.
* `copy_options` - (Optional) Copy options for copying the data from the s3 intermediate bucket into redshift, for example to change the default delimiter. For valid values, see the [AWS documentation](http://docs.aws.amazon.com/firehose/latest/APIReference/API_CopyCommand.html)
* `data_table_columns` - (Optional) The data table columns that will be targeted by the copy command.
* `cloudwatch_logging_options` - (Optional) The CloudWatch Logging Options for the delivery stream. More details are given below

The `elasticsearch_configuration` object supports the following:

* `buffering_interval` - (Optional) Buffer incoming data for the specified period of time, in seconds between 60 to 900, before delivering it to the destination.  The default value is 300s.
* `buffering_size` - (Optional) Buffer incoming data to the specified size, in MBs between 1 to 100, before delivering it to the destination.  The default value is 5MB.
* `domain_arn` - (Required) The ARN of the Amazon ES domain.  The IAM role must have permission for `DescribeElasticsearchDomain`, `DescribeElasticsearchDomains`, and `DescribeElasticsearchDomainConfig` after assuming `RoleARN`.  The pattern needs to be `arn:.*`.
* `index_name` - (Required) The Elasticsearch index name.
* `index_rotation_period` - (Optional) The Elasticsearch index rotation period.  Index rotation appends a timestamp to the IndexName to facilitate expiration of old data.  Valid values are `NoRotation`, `OneHour`, `OneDay`, `OneWeek`, and `OneMonth`.  The default value is `OneDay`.
* `retry_duration` - (Optional) After an initial failure to deliver to Amazon Elasticsearch, the total amount of time, in seconds between 0 to 7200, during which Firehose re-attempts delivery (including the first attempt).  After this time has elapsed, the failed documents are written to Amazon S3.  The default value is 300s.  There will be no retry if the value is 0.
* `role_arn` - (Required) The ARN of the IAM role to be assumed by Firehose for calling the Amazon ES Configuration API and for indexing documents.  The pattern needs to be `arn:.*`.
* `s3_backup_mode` - (Optional) Defines how documents should be delivered to Amazon S3.  Valid values are `FailedDocumentsOnly` and `AllDocuments`.  Default value is `FailedDocumentsOnly`.
* `type_name` - (Required) The Elasticsearch type name with maximum length of 100 characters.
* `cloudwatch_logging_options` - (Optional) The CloudWatch Logging Options for the delivery stream. More details are given below

The `cloudwatch_logging_options` object supports the following:

* `enabled` - (Optional) Enables or disables the logging. Defaults to `false`.
* `log_group_name` - (Optional) The CloudWatch group name for logging. This value is required if `enabled` is true.
* `log_stream_name` - (Optional) The CloudWatch log stream name for logging. This value is required if `enabled` is true.

## Attributes Reference

* `arn` - The Amazon Resource Name (ARN) specifying the Stream

[1]: https://aws.amazon.com/documentation/firehose/
