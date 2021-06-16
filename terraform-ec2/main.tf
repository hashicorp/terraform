# Terraform Settings Block
terraform {
    required_version = "~> 1.0.0"
    required_providers {
      aws = {
          source  = "hashicorp/aws"
          version = "~> 3.0"
      }
    random = {
      source = "hashicorp/random"
      version = "~> 3.1.0"
    }
    }
}

# Provider Block
provider "aws" {
    profile = "ops-lab"
    region  = "us-east-1"
}

# Resource Block
resource "aws_instance" "webserver" {
    ami                                  = "ami-01cc041b5d0b989b8"
    instance_type                        = "t2.micro"
    associate_public_ip_address          = true
    instance_initiated_shutdown_behavior = "terminate"
    subnet_id                            = "subnet-078726d8c8b2665e1"
    user_data                            = "${file("install_httpd.sh")}"
    tags = {
        Name = "webserver"
        Role = "ops-lab"
    }

}
