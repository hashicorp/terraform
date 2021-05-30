resource "aws_instance" "foo" {
  num     = "2"
  compute = "list.#"
}

resource "aws_instance" "bar" {
  foo = aws_instance.foo.list.0
}
