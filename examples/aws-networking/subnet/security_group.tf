resource "aws_security_group" "az" {
  name        = "az-${data.aws_availability_zone.target.name}"
  description = "Open access within the AZ ${data.aws_availability_zone.target.name}"
  vpc_id      = "${var.vpc_id}"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = -1
    cidr_blocks = ["${aws_subnet.main.cidr_block}"]
  }
}
