variable "public_key_path" {
  description = <<DESCRIPTION
Path to the SSH public key to be used for authentication.
Ensure this keypair is added to your local SSH agent so provisioners can
connect.

Example: ~/.ssh/terraform.pub
DESCRIPTION
}

variable "key_name" {
  description = "Desired name of AWS key pair"
}

variable "aws_region" {
  description = "AWS region to launch servers."
  default = "us-west-2"
}

# Ubuntu Trusty 14.04 LTS (x64)
# (ubuntu/images/hvm-ssd/ubuntu-trusty-14.04-amd64-server-20160714)
variable "aws_amis" {
  default = {
    eu-west-1 = "ami-ed82e39e"
    us-east-1 = "ami-3bdd502c"
    us-west-1 = "ami-48db9d28"
    us-west-2 = "ami-d732f0b7"
  }
}
