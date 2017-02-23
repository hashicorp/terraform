resource "aws_instance" "a" {}
resource "aws_instance" "b" {
    value = "${aws_instance.a.id}"
}

resource "aws_instance" "c" {
    value = "${aws_instance.b.id}"
}
