resource "aws_vpc" "me" {}

resource "aws_subnet" "me" {
  vpc_id = "${aws_vpc.me.id}"
}

resource "aws_instance" "me" {
  subnet_id = "${aws_subnet.me.id}"
}

resource "aws_vpc" "notme" {}
resource "aws_subnet" "notme" {}
resource "aws_instance" "notme" {}
resource "aws_instance" "notmeeither" {
  name = "${aws_instance.me.id}"
}
