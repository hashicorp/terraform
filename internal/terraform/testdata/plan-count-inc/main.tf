resource "aws_instance" "foo" {
    foo = "foo"
    count = 3
}

resource "aws_instance" "bar" {
    foo = "bar"
}
