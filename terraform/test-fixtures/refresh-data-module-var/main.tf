resource "aws_instance" "A" {
    foo = "bar"
}

module "child" {
    source = "child"
    key    = "${aws_instance.A.id}"
}
