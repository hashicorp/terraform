list "test_instance" "example" {
  provider = test

  config {
    ami = var.target_ami
    foo = var.instance_name
  }
}
