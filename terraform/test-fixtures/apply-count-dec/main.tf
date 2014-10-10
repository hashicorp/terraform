resource "aws_instance" "foo" {
    foo = "foo"
    count = 2
}

resource "aws_instance" "bar" {
    foo = "bar"
}
