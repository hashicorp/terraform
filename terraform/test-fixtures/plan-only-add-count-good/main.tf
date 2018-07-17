resource "aws_instance" "foo" {
  count = "2"
  current = "${count.index}"

  lifecycle {
    only_add = true
  }
}
