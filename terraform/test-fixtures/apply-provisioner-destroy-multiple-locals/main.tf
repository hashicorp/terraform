locals {
  value = "local"
  foo_id = "${aws_instance.foo.id}"

  // baz is not in the state during destroy, but this is a valid config that
  // should not fail.
  baz_id = "${aws_instance.baz.id}"
}

resource "aws_instance" "baz" {}

resource "aws_instance" "foo" {
    provisioner "shell" {
        command  = "${local.value}"
        when = "destroy"
    }
}

resource "aws_instance" "bar" {
    provisioner "shell" {
        command  = "${local.foo_id}"
        when = "destroy"
    }
}
