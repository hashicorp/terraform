module "child" {
    source = "./child"
}

provider "aws" {
    from = "root"
}

resource "aws_instance" "foo" {
    from = "root"
}
