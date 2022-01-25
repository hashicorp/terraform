variable "key" {}

data "aws_data_source" "foo" {
  id = "${var.key}"
}

