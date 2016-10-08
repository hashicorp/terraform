variable "count" {}

output "instance_ids" {
    type = "list"
    value = "${aws_instance.foo.*.id}"
}

resource "aws_instance" "foo" {
    count = "${var.count}"
    foo = "foo"
}
