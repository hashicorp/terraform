resource "aws_instance" "foo" {
  num = "3"
}

resource "aws_instance" "bar" {
  num = aws_instance.foo.num
}
