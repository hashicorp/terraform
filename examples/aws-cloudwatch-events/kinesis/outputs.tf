output "rule_arn" {
  value = "${aws_cloudwatch_event_rule.foo.arn}"
}

output "kinesis_stream_arn" {
  value = "${aws_kinesis_stream.foo.arn}"
}
