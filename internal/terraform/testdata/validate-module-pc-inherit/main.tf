module "child" {
    source = "./child"
}

provider "aws" {
    set = true
}

resource "aws_instance" "foo" {}
