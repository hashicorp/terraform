terraform {
  required_providers {
    your-aws = {
      source = "hashicorp/aws"
    }
  }
}

resource "aws_instance" "web" {
  provider = "your-aws"
}
