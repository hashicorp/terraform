provider "aws" {
  region = "${var.aws_region}"
}

resource "aws_cloudwatch_event_rule" "foo" {
  name = "${var.rule_name}"
  event_pattern = <<PATTERN
{
  "detail-type": [
    "AWS API Call via CloudTrail"
  ],
  "detail": {
    "eventSource": [
      "autoscaling.amazonaws.com"
    ]
  }
}
PATTERN
  role_arn = "${aws_iam_role.role.arn}"
}

resource "aws_iam_role" "role" {
	name = "${var.iam_role_name}"
	assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "events.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
POLICY
}

resource "aws_iam_role_policy" "policy" {
  name = "tf-example-policy"
  role = "${aws_iam_role.role.id}"
  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "kinesis:PutRecord",
        "kinesis:PutRecords"
      ],
      "Resource": [
        "*"
      ],
      "Effect": "Allow"
    }
  ]
}
POLICY
}

resource "aws_cloudwatch_event_target" "foobar" {
	rule = "${aws_cloudwatch_event_rule.foo.name}"
	target_id = "${var.target_name}"
	arn = "${aws_kinesis_stream.foo.arn}"
}

resource "aws_kinesis_stream" "foo" {
  name = "${var.stream_name}"
  shard_count = 1
}
