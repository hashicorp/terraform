variable "num" {}

resource "aws_instance" "bar" {
    foo = "bar${count.index}"
    count = "${var.num}"
}

output "output" {
    value = "${join(",", aws_instance.bar.*.foo)}"
}
