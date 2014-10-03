resource "aws_instance" "foo" {
    foo = "${count.index}"
}
