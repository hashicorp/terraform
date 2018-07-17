resource "aws_instance" "foo" {
  count = "1"
  current = "${count.index+1}"

  lifecycle {
     only_add = true
  }
}
