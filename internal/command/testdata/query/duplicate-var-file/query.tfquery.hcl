variable "target_ami" {
  description = "The AMI to search for"
  type        = string
}

list "test_instance" "example" {
  provider = test

  config {
    ami = var.target_ami
  }
}
