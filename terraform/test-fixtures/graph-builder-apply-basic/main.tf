module "child" {
    source = "./child"
}

resource "aws_instance" "create" {}

resource "aws_instance" "other" {
    foo = "${aws_instance.create.bar}"
}
