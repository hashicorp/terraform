resource "aws_instance" "web" {
    lifecycle {
        create_before_destroy = true
    }
}

resource "aws_load_balancer" "lb" {
    member = "${aws_instance.web.id}"
}
