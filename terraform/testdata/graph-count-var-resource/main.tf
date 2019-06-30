resource "aws_instance" "foo" {}

resource "aws_instance" "web" {
    count = "${aws_instance.foo.bar}"
}

resource "aws_load_balancer" "weblb" {
    members = "${aws_instance.web.*.id}"
}
