variable "aws_region" {
  description = "The AWS region to create things in."
  default     = "us-west-2"
}

# Ubuntu Xenial 16.04 LTS (x64)
variable "aws_amis" {
  default = {
    "eu-west-1" = "ami-405f7226"
    "us-east-1" = "ami-f4cc1de2"
    "us-west-1" = "ami-16efb076"
    "us-west-2" = "ami-a58d0dc5"
  }
}
