module "child" {
    input = "42"
    source = "./child"
}

resource "aws_instance" "bar" {
    foo = "2"
}
