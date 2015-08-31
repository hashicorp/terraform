variable "aws_region" {
  description = "The AWS region to create resources in."
  default = "us-east-1"
}

# AMI's from http://cloud-images.ubuntu.com/locator/ec2/
variable "aws_amis" {
  default = {
    eu-west-1 =  "ami-b1cf19c6"
    us-east-1 =  "ami-de7ab6b6"
    us-west-1 =  "ami-3f75767a"
    us-west-2 =  "ami-21f78e11"
  }
}
