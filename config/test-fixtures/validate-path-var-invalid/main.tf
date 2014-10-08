resource "aws_instance" "foo" {
    foo = "${path.nope}"
}
