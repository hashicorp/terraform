module "child" {
    source = "./child"
}

resource "aws_instance" "foo" {
    foo = "bar"
}

resource "aws_instance" "bar" {
    foo = "bar"
}
