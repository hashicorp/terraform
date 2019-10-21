resource "aws_instance" "web" {
    count = 3
}

resource "aws_load_balancer" "weblb" {
    members = "${aws_instance.web.*.id}"
}
