variable "target_ami" {
  description = "The AMI to search for"
  type        = string
  validation {
    condition     = length(var.target_ami) > 10
    error_message = "AMI ID must be longer than 10 characters."
  }
}

list "test_instance" "example" {
  provider = test

  config {
    ami = var.target_ami
    foo = "invalid-instance"
  }
}
