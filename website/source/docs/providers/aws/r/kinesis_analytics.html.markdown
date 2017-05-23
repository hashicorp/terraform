---
layout: "aws"
page_title: "AWS: aws_kinesis_analytics"
sidebar_current: "docs-aws-resource-kinesis-analytics"
description: |-
  Provides an AWS Kinesis Analytics Application
---

# aws\_kinesis\_analytics

Provides a Kinesis Analytics resource. The service enables you to quickly author and run powerful SQL code against streaming sources to perform time series analytics, feed real-time dashboards, and create real-time metrics.

For more details, see the [Amazon Kinesis Analytics Documentation][2].

## Example Minimum Usage
_these are the minimum required inputs to create an analyics app_

```hcl
resource "aws_kinesis_analytics" "test_application" {
  name = "terraform-kinesis-analytics-test"
  application_description = "test description"
  application_code = "SELECT 1\n"
}

```
## Example with streams 
_these are the minimum required dependencies and syntax for declaring a fully configured application_

```hcl
resource "aws_iam_role" "ka_test_role" {
  name = "terraform-kinesis-analytics-test-role"
  description = "this role has no attached policy. it is just for testing instantiation kinesis analytics connections to other resources onCreate!"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "kinesisanalytics.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_kinesis_stream" "test_input_stream_a" {
  name             = "terraform-kinesis-analytics-input-a-test"
  shard_count      = 1
}
resource "aws_kinesis_stream" "test_input_stream_b" {
  name             = "terraform-kinesis-analytics-input-b-test"
  shard_count      = 1
}

resource "aws_kinesis_stream" "test_output_stream_a" {
  name             = "terraform-kinesis-analytics-output-a-test"
  shard_count      = 1
}
resource "aws_kinesis_stream" "test_output_stream_b" {
  name             = "terraform-kinesis-analytics-output-b-test"
  shard_count      = 1
}

resource "aws_kinesis_analytics" "test_application" {
  name = "terraform-kinesis-analytics-test"
  application_description = "test description"
  application_code = "SELECT 1\n"
  inputs{
    name = "SOURCE_SQL_STREAM_A"
    record_format_type = "JSON"
    record_format_encoding = "UTF-8"
    record_row_path = "$"
    columns{
      name = "id"
      sql_type = "INTEGER"
      mapping = "id"
    }
    columns{
      name = "firstName"
      sql_type = "VARCHAR(256)"
      mapping = "firstName"
    }
    arn = "${aws_kinesis_stream.test_input_stream_a.arn}"
    role_arn = "${aws_iam_role.ka_test_role.arn}"
  }
  inputs{
    name = "SOURCE_SQL_STREAM_B"
    record_format_type = "CSV"
    record_format_encoding = "UTF-8"
    record_row_delimiter = "\n"
    record_column_delimiter = ","
    columns{
      name = "id"
      sql_type = "INTEGER"
      mapping = "id"
    }
    columns{
      name = "lastName"
      sql_type = "VARCHAR(256)"
      mapping = "lastName"
    }
    arn = "${aws_kinesis_stream.test_input_stream_b.arn}"
    role_arn = "${aws_iam_role.ka_test_role.arn}"
  }
  outputs {
    name = "DESTINATION_SQL_STREAM_A"
    record_format_type = "JSON"
    arn = "${aws_kinesis_stream.test_output_stream_a.arn}"
    role_arn = "${aws_iam_role.ka_test_role.arn}"
  }
  outputs {
    name = "DESTINATION_SQL_STREAM_B"
    record_format_type = "CSV"
    arn = "${aws_kinesis_stream.test_output_stream_b.arn}"
    role_arn = "${aws_iam_role.ka_test_role.arn}"
  }
}
```

## Argument Reference
_The following arguments are supported:_

- `name` - (Required) A name to identify the application. This is unique to the AWS account and region the application is created in.
- `application_description` - (Optional) a short sring that identifies the application purpose.
- `application_code` - (Optional) sql code that will run at the heart of this application.
- `inputs` - (Optional) up to four kinesis streams are supported as stream inputs.
- `outputs` - (Optional) up to four kinesis streams/firehose are supported as stream outputs.


## Attributes Reference

- `id` - The unique Stream id
- `name` - The unique application name
- `create_timestamp` - this timestamp is required to delete a Kinesis Analytics Application
- `arn` - The Amazon Resource Name (ARN) specifying the application (same as `id`)


## Writing Application Code
_the application\_code attribute is sql code that *is* your application code_
- For more details, see the [Analytics SQL Reference][1].


## Import
_not supported yet. coming soon!_


[1]: http://docs.aws.amazon.com/kinesisanalytics/latest/sqlref/analytics-sql-reference.html
[2]: http://docs.aws.amazon.com/kinesisanalytics/latest/dev/what-is.html
[3]: https://docs.aws.amazon.com/kinesisanalytics/latest/dev/example-apps.html