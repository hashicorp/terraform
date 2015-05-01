module "child" {
    source = "./child"
    var = "${aws_instance.parent.value}"
}

resource "aws_instance" "parent" {
    value = "foo"
}
