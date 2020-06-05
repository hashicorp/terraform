provider "aws" {}

module "child" {
    source = "./child"
}
