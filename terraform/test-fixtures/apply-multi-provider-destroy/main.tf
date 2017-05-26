resource "vault_instance" "foo" {}

provider "aws" {
  addr = "${vault_instance.foo.id}"
}

resource "aws_instance" "bar" {
    foo = "bar"
}
