module "a" {
    source = "./child"
    in = "${aws_instance.b.id}"
}

resource "aws_instance" "b" {}

resource "aws_instance" "c" {
    some_input = "${module.a.out}"

    depends_on = ["aws_instance.b"]
}
