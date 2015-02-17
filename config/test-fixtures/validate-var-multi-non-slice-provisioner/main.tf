resource "aws_instance" "foo" {
    count = 3
}

resource "aws_instance" "bar" {
    provisioner "local-exec" {
        foo = "${aws_instance.foo.*.id}"
    }
}
