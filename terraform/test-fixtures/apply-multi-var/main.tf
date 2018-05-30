variable "num" {}

resource "aws_instance" "bar" {
    count = "${var.num}"
    foo = "bar${count.index}"
}

output "output" {
    value = "${join(",", aws_instance.bar.*.foo)}"
}
