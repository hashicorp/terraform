variable "services" {
  type = list(map(string))
}

resource "aws_instance" "inner2" {
  looked_up = var.services[0]["elb"]
}

