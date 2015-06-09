resource "aws_security_group" "default" {
  name = "main_rds_sg"
  description = "Allow all inbound traffic"

  ingress {
      from_port = 0
      to_port = 65535
      protocol = "TCP"
      cidr_blocks = ["${var.cidr_blocks}"]
  }

  egress {
      from_port = 0
      to_port = 0
      protocol = "-1"
      cidr_blocks = "0.0.0.0/0"
  }

  tags {
    Name = "${var.sg_name}"
  }
}
