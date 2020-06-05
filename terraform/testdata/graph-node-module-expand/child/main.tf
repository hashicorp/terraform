resource "aws_instance" "foo" {}
resource "aws_instance" "bar" {
    var = "${aws_instance.foo.whatever}"
}
