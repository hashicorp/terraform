resource "aws_instance" "bar" {
    key = "hello"
}

resource "aws_instance" "foo" {
    foo = "bar"

    provisioner "shell" {
        foo  = "${aws_instance.bar.key}"
        when = "destroy"
    }
}
