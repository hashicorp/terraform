variable "count" {
  default = 15
}

resource "aws_instance" "bar" {
  count = "${var.count}"
  foo   = "index-${count.index}"
}

resource "aws_instance" "baz" {
  count = "${var.count}"
  foo   = "baz-${element(aws_instance.bar.*.foo, count.index)}"
}

output "should-be-11" {
  value = "${element(aws_instance.baz.*.foo, 11)}"
}
