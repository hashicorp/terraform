resource "aws_instance" "foo" {
  count = 3
}

resource "aws_instance" "bar" {
  for_each = { for idx, instance in aws_instance.foo : idx => instance }
  foo = "${each.value.id}"
}
