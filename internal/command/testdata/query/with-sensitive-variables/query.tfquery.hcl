variable "target_ami" {
  description = "The AMI to search for"
  type        = string
  default     = "ami-12345"
}

variable "sensitive_foo" {
  description = "a"
  type        = string
  sensitive   = true
}

list "test_instance" "example" {
  provider = test

  config {
    ami = var.target_ami
    foo = var.sensitive_foo
  }
}
