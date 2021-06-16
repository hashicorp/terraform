resource "aws_vpc" "labvpc" {
  cidr_block = "10.10.10.0/24"
  enable_dns_support = true
  enable_dns_hostnames = true
  instance_tenancy = "default"
  tags = {
      Name = "lab-vpc"
  }
}