module "child" {
    source = "./child"
    var = "${aws_instance.parent.value}"
}

resource "aws_instance" "parent" {
    value = "foo"
}

resource "aws_instance" "parent-output" {
    value = "${module.child.output}"
}
