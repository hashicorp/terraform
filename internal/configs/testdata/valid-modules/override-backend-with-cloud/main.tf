mnptu {
  backend "foo" {
    path = "relative/path/to/mnptu.tfstate"
  }
}

resource "aws_instance" "web" {
  ami = "ami-1234"
  security_groups = [
    "foo",
    "bar",
  ]
}
