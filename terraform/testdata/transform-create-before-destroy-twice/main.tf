resource "aws_lc" "foo" {
    lifecycle { create_before_destroy = true }
}

resource "aws_autoscale" "bar" {
    lc = "${aws_lc.foo.id}"

    lifecycle { create_before_destroy = true }
}
