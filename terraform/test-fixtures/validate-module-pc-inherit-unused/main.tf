module "child" {
    source = "./child"
}

provider "aws" {
    foo = "set"
}
