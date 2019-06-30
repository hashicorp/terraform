variable "foo" {
  default = "${aws_instance.foo.bar}"
}
