terraform {
  required_providers {
    my_aws = {
      source = "hashicorp/aws"
    }
  }
}

resource "aws_instance" "web" {
  provider = "my_aws"
}
