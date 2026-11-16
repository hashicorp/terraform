provider "aws" {
  access_key = "abc123"
}

module "child" {
  source = "./child"
}
