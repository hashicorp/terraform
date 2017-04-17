resource "aws_instance" "foo" {
    compute = "foo"
}

module "child" {
    source = "./child"
    value = "${aws_instance.foo.foo}"
}
