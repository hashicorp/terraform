provider "aws" {}

resource "aws_instance" "foo" {}

check "aws_instance_exists" {
  data "aws_data_source" "bar" {
    id = "baz"
  }

  assert {
    condition     = data.aws_data_source.bar.foo == "Hello, world!"
    error_message = "incorrect value"
  }
}
