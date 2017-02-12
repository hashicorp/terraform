variable "count" { default = 15 }

resource "aws_instance" "bar" {
    count = "${var.count}"
    foo = "index-${count.index}"
}

output "should-be-11" {
    value = "${element(aws_instance.bar.*.foo, 11)}"
}
