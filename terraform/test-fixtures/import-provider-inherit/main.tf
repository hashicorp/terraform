provider "aws" {
  foo = "bar"
}

module "child" {
    source = "./child"
    providers {
        "aws" = "aws"
    }
}
