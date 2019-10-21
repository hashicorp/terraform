resource "aws_vpc" "notme" {}

resource "aws_subnet" "notme" {
  depends_on = [
    aws_vpc.notme,
  ]
}

resource "aws_instance" "me" {
  depends_on = [
    aws_subnet.notme,
  ]
}

resource "aws_instance" "notme" {}
resource "aws_instance" "metoo" {
  depends_on = [
    aws_instance.me,
  ]
}

resource "aws_elb" "me" {
  depends_on = [
    aws_instance.me,
  ]
}
