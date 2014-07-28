resource "aws_instance" "foo" {
  ami = "${aws_instance.bar.id}"
}

resource "aws_instance" "bar" {
  ami = "${aws_instance.foo.id}"
}
