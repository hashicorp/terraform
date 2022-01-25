module "child" {
    source = "./child"
}

resource "aws_instance" "bar" {
    foo = "bar"
}
