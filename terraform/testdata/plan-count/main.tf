resource "aws_instance" "foo" {
    count = 5
    foo = "foo"
}

resource "aws_instance" "bar" {
    foo = "${join(",", aws_instance.foo.*.foo)}"
}
