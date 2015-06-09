variable "subnet_1_cidr" {
	default = "10.0.1.0/24"
    description = "Your AZ"
}

variable "subnet_2_cidr" {
	default = "10.0.2.0/24"
    description = "Your AZ"
}

variable "az_1" {
	default = "us-east-1b"
    description = "Your AZ"
}

variable "az_2" {
	default = "us-east-1c"
    description = "Your AZ"
}

variable "vpc_id" {
	default = "vpc-b6090dd3"
    description = "Your VPC ID"
}