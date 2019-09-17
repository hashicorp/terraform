variable "list" {
  default = ["1", "2"]
}

resource "aws_instance" "a" {
  count = "${length(var.list)}"
}

output "out" {
  value = "${aws_instance.a.*.id}"
}
