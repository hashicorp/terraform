module "child" {
    source = "./child"
}

resource "aws_instance" "a" {
    depends_on = ["module.child"]
}
