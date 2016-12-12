resource "aws_instance" "foo" {
	count = 2
    id = "foo-${count.index}"
	foo = "ok"
}

resource "aws_instance" "bar" {
	count = 2
    id = "bar-${count.index}"
    foo = "bar-${element(aws_instance.foo.*.id, count.index)}"
}
