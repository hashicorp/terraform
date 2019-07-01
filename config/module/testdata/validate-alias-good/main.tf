provider "aws" { alias = "foo" }

module "child" {
    source = "./child"
}
