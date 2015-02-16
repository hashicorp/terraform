provider "aws" {}

resource "aws_lc" "foo" {}

resource "aws_asg" "foo" {
    lc = "${aws_lc.foo.id}"

    lifecycle { create_before_destroy = true }
}
