resource "vault_instance" "foo" {}

provider "aws" {
  value = "${vault_instance.foo.id}"
}

module "child" {
  source = "./child"
}
