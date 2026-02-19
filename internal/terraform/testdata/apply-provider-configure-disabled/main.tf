provider "aws" {
    foo = "bar"
}

module "child" {
    source = "./child"
}
