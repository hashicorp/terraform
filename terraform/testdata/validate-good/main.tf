resource "aws_instance" "foo" {
    num = "2"
    foo = "bar"
}

resource "aws_instance" "bar" {
    foo = "bar"
}
