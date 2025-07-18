locals {
  ami = "ami-12345"
}

list "test_instance" "example" {
  provider = test

  config {
    ami = local.ami
  }
}
