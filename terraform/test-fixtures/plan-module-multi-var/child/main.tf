variable "things" {}

resource "aws_instance" "bar" {
  baz = "baz"
  count = 2
}

resource "aws_instance" "foo" {
  foo = "${join(",",aws_instance.bar.*.baz)}"
}
