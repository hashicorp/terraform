resource "aws_vpc" "notme" {}

resource "aws_subnet" "notme" {
  vpc_id = "${aws_vpc.notme.id}"
}

resource "aws_instance" "me" {
  subnet_id = "${aws_subnet.notme.id}"
}

resource "aws_instance" "notme" {}
resource "aws_instance" "metoo" {
  name = "${aws_instance.me.id}"
}

resource "aws_elb" "me" {
  instances = "${aws_instance.me.*.id}"
}
