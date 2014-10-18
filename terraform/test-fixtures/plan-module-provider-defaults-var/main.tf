module "child" {
    source = "./child"
}

provider "aws" {
    from = "${var.foo}"
}
