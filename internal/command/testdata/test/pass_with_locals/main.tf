resource "test_instance" "foo" {
  ami = "bar"
}

locals {
  value = test_instance.foo.ami
}
