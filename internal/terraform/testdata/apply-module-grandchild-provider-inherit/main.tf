provider "aws" {
  value = "foo"
}

module "child" {
  source = "./child"
}
