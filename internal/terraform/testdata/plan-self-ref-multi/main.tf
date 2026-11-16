resource "aws_instance" "web" {
    foo = "${aws_instance.web.0.foo}"
    count = 4
}
