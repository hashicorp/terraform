module "grandchild" {
    source = "./child"
}

resource "aws_instance" "b" {
    depends_on = ["module.grandchild"]
}
