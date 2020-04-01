terraform {
  required_providers {
    your_aws = {
      source = "hashicorp/aws"
    }
  }
}

resource "aws_instance" "web" {
  provider = "your_aws"
}
