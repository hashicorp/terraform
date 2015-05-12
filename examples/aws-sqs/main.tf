# Specify the provider and access details
provider "aws" {
    region = "${var.aws_region}"
}

resource "aws_sqs_queue" "terraform_queue" {
  queue = "terraform-example-renamed"
}

resource "aws_sqs_queue" "terrform_queue_attr" {
  queue = "terraform-example-attr"
  delay_seconds = 90
  max_message_size = 2048
  message_retention_seconds = 86400
  receive_wait_time_seconds = 10
}

resource "aws_sqs_queue" "terraform_queue_too" {
	queue = "terraform-queue-too"
	delay_seconds = 120
	max_message_size = 4096
}
