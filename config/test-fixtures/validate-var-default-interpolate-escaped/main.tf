variable "foo" {
  # This should be considered valid since the sequence is escaped and is
  # thus not actually an interpolation.
  default = "foo bar $${aws_instance.foo.bar}"
}
