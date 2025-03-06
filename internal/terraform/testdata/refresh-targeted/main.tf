resource "aws_vpc" "metoo" {}
resource "aws_instance" "notme" { }
resource "aws_instance" "me" {
  vpc_id = "${aws_vpc.metoo.id}"
}
resource "aws_elb" "meneither" {
  instances = toset([for instance in aws_instance.me : instance])
}
