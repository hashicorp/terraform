variable "aws_region" {
  description = "The AWS region to create resources in."
  default     = "us-east-1"
}

variable "rule_name" {
  description = "The name of the CloudWatch Event Rule"
  default     = "tf-example-cloudwatch-event-rule-for-sns"
}

variable "target_name" {
  description = "The name of the CloudWatch Event Target"
  default     = "tf-example-cloudwatch-event-target-for-sns"
}

variable "sns_topic_name" {
  description = "The name of the SNS Topic to send events to"
  default     = "tf-example-sns-topic"
}
