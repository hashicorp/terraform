module "foo" {
    source = "./bar"
}

resource "aws_instance" "web" {
    count = "${module.foo.bar}"
}
