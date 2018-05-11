provider "aws" {
  foo = "bar"
}

resource "aws_instance" "test" {
  provisioner "shell" {
    test_string = "foo"
  }
}
