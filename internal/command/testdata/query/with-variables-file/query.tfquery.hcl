variable "target_ami" {
  description = "The AMI to search for"
  type        = string
}

variable "environment" {
  description = "The environment tag"
  type        = string
  default     = "test"
}

variable "instance_count" {
  description = "Number of instances to find"
  type        = number
  default     = 2
}

list "test_instance" "example" {
  provider = test

  config {
    ami = var.target_ami
    foo = var.environment
  }
}
