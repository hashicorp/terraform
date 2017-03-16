variable "instance_id" {
}

output "instance_id" {
  value = "${var.instance_id}"
}

resource "aws_instance" "foo" {
  foo = "${var.instance_id}"
}
