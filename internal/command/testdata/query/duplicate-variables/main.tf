terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

variable "target_ami" {
  description = "The AMI to search for"
  type        = string
  default     = "ami-12345"
}

variable "instance_name" {
  description = "The instance name to search for"
  type        = string
}

provider "test" {}

resource "test_instance" "example" {
  ami = "ami-12345"
}
