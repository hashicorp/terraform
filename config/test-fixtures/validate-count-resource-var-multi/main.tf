resource "aws_instance" "foo" {}

resource "aws_instance" "web" {
    count = "${length(aws_instance.foo.*.bar)}"
}
