resource "aws_instance" "foo" {
    count = 2
    compute = "ip.#"
}

resource "aws_instance" "bar" {
    count = 1
    foo = "${aws_instance.foo.*.ip[count.index]}"
}
