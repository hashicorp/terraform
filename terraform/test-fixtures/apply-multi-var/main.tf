variable "count" {}

resource "aws_instance" "bar" {
    foo = "bar${count.index}"
    count = "${var.count}"
}

output "output" {
    value = "${join(",", aws_instance.bar.*.foo)}"
}
