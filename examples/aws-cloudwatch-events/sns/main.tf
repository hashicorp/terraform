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
      "cloudtrail.amazonaws.com"
    ]
  }
}
PATTERN
}

resource "aws_cloudwatch_event_target" "bar" {
  rule      = "${aws_cloudwatch_event_rule.foo.name}"
  target_id = "${var.target_name}"
  arn       = "${aws_sns_topic.foo.arn}"
}

resource "aws_sns_topic" "foo" {
  name = "${var.sns_topic_name}"
}
