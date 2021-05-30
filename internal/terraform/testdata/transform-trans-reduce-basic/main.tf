resource "aws_instance" "A" {}

resource "aws_instance" "B" {
    A = "${aws_instance.A.id}"
}

resource "aws_instance" "C" {
    A = "${aws_instance.A.id}"
    B = "${aws_instance.B.id}"
}
