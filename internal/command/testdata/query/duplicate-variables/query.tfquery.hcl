// already defined in main.tf
variable "instance_name" {
  description = "The instance name to search for"
  type        = string
}


list "test_instance" "example" {
  provider = test

  config {
    ami = var.target_ami
    foo = var.instance_name
  }
}
