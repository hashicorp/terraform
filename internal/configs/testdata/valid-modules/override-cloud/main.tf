terraform {
  cloud {
    organization = "foo"
    should_not_be_present_with_override = true
  }
}

resource "aws_instance" "web" {
  ami = "ami-1234"
  security_groups = [
    "foo",
    "bar",
  ]
}
