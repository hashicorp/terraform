resource "aws_instance" "foo" {
    count = 2
    foo = "foo"
}
