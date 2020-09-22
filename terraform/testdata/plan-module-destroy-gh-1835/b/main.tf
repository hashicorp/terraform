variable "a_id" {}

resource "aws_instance" "b" {
  command = "echo ${var.a_id}"
}
