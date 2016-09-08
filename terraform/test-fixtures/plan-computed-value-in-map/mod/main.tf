variable "services" {
  type = "list"
}

resource "aws_instance" "inner2" {
  looked_up = "${lookup(var.services[0], "elb")}"
}

