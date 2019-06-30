resource "aws_lc" "foo" {}

resource "aws_asg" "bar" {
  lc = "${aws_lc.foo.id}"
}
